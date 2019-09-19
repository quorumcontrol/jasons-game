package devink

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	uuid "github.com/hashicorp/go-uuid"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/devink/devnet"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("devink")

type DevInkSource struct {
	ChainTree *consensus.SignedChainTree
	Net       devnet.DevRemoteNetwork
}

func inkSendTxId() (string, error) {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return "", errors.Wrap(err, "error generating dev ink send transaction ID")
	}

	return id, nil
}

func NewSource(ctx context.Context, dataStoreDir string, connectToLocalnet bool) (*DevInkSource, error) {
	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, connectToLocalnet)
	if err != nil {
		return nil, errors.Wrap(err, "error setting up notary group")
	}

	dsDir := config.EnsureExists(dataStoreDir)

	ds, err := config.LocalDataStore(dsDir.Path)
	if err != nil {
		return nil, errors.Wrap(err, "error setting up local data store")
	}

	signingKey, err := network.GetOrCreateStoredPrivateKey(ds)
	if err != nil {
		return nil, errors.Wrap(err, "error getting private signingKey")
	}
	fmt.Printf("INK_TREE_KEY='%s'\n", base64.StdEncoding.EncodeToString(crypto.FromECDSA(signingKey)))

	netCfg := &network.RemoteNetworkConfig{
		NotaryGroup:   notaryGroup,
		KeyValueStore: ds,
		SigningKey:    signingKey,
	}

	rNet, err := network.NewRemoteNetworkWithConfig(ctx, netCfg)
	if err != nil {
		return nil, errors.Wrap(err, "error setting up network")
	}

	net := devnet.DevRemoteNetwork{RemoteNetwork: rNet}

	devInkSource, err := net.FindOrCreatePassphraseTree("dev-ink-source")
	if err != nil {
		return nil, errors.Wrap(err, "error getting ink source chaintree")
	}

	return &DevInkSource{ChainTree: devInkSource, Net: net}, nil
}

func (dis *DevInkSource) tokenLedger(ctx context.Context) (*consensus.TreeLedger, error) {
	devInkChainTree := dis.ChainTree
	devInkSourceTree, err := devInkChainTree.ChainTree.Tree(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting ink source tree")
	}

	devInkTokenName := &consensus.TokenName{ChainTreeDID: devInkChainTree.MustId(), LocalName: "ink"}

	return consensus.NewTreeLedger(devInkSourceTree, devInkTokenName), nil
}

func (dis *DevInkSource) EnsureToken(ctx context.Context) error {
	devInkChainTree := dis.ChainTree

	devInkLedger, err := dis.tokenLedger(ctx)
	if err != nil {
		return errors.Wrap(err, "error creating dev ink token ledger")
	}

	devInkExists, err := devInkLedger.TokenExists()
	if err != nil {
		return errors.Wrap(err, "error checking for existence of dev ink")
	}
	log.Info("dev ink token already exists")

	if !devInkExists {
		log.Info("establishing new dev ink token")

		establishInk, err := chaintree.NewEstablishTokenTransaction("ink", 0)
		if err != nil {
			return errors.Wrap(err, "error creating establish_token transaction for dev ink")
		}

		devInkChainTree, err = dis.Net.PlayTransactions(devInkChainTree, []*transactions.Transaction{establishInk})
		if err != nil {
			return errors.Wrap(err, "error establishing dev ink token")
		}

		dis.ChainTree = devInkChainTree
	}

	return nil
}

func (dis *DevInkSource) EnsureBalance(ctx context.Context, minimum uint64) error {
	devInkChainTree := dis.ChainTree

	devInkLedger, err := dis.tokenLedger(ctx)
	if err != nil {
		return errors.Wrap(err, "error creating dev ink token ledger")
	}

	devInkBalance, err := devInkLedger.Balance()
	if err != nil {
		return errors.Wrap(err, "error getting dev ink balance")
	}

	log.Debugf("devink balance: %d", devInkBalance)

	if devInkBalance < minimum {
		log.Infof("devink balance (%d) is lower than minimum (%d); minting some more", devInkBalance, minimum)

		mintInk, err := chaintree.NewMintTokenTransaction("ink", minimum)
		if err != nil {
			return errors.Wrap(err, "error creating mint_token transaction for dev ink")
		}

		devInkChainTree, err = dis.Net.PlayTransactions(devInkChainTree, []*transactions.Transaction{mintInk})
		if err != nil {
			return errors.Wrap(err, "error minting dev ink token")
		}

		dis.ChainTree = devInkChainTree
	}

	return nil
}

func (dis *DevInkSource) SendInk(ctx context.Context, destinationChainId string, amount uint64) (*transactions.TokenPayload, error) {
	devInkChainTree := dis.ChainTree
	devInkDID := devInkChainTree.MustId()

	tokenName, err := consensus.CanonicalTokenName(devInkChainTree.ChainTree.Dag, devInkDID, "ink", true)
	if err != nil {
		return nil, errors.Wrap(err, "error getting canonical token name for dev ink")
	}

	sendTxId, err := inkSendTxId()
	if err != nil {
		return nil, errors.Wrap(err, "error generating ink send transaction id")
	}

	sendInk, err := chaintree.NewSendTokenTransaction(sendTxId, tokenName.String(), amount, destinationChainId)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating send_token transaction for dev ink to %s", destinationChainId)
	}

	devInkSource, txResp, err := dis.Net.PlayTransactionsWithResp(devInkChainTree, []*transactions.Transaction{sendInk})
	if err != nil {
		return nil, errors.Wrapf(err, "error sending dev ink token to %s", destinationChainId)
	}

	tokenSend, err := consensus.TokenPayloadForTransaction(devInkSource.ChainTree, tokenName, sendTxId, &txResp.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "error generating dev ink token payload")
	}

	return tokenSend, nil
}
