package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-uuid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/devink/devnet"
	"github.com/quorumcontrol/jasons-game/network"
)

const (
	dataStoreDir = "dev-ink"
	amount       = 10000
)

func inkSendTxId() string {
	id, err := uuid.GenerateUUID()
	if err != nil {
		panic(errors.Wrap(err, "error generating dev ink send transaction ID"))
	}

	return id
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, true)
	if err != nil {
		panic(fmt.Errorf("error setting up local notary group: %v", err))
	}

	dsDir := config.EnsureExists(dataStoreDir)

	ds, err := config.LocalDataStore(dsDir.Path)
	if err != nil {
		panic(fmt.Errorf("error setting up local data store: %v", err))
	}

	signingKey, err := network.GetOrCreateStoredPrivateKey(ds)
	fmt.Println("signing key:", signingKey)
	if err != nil {
		panic(errors.Wrap(err, "error getting private signingKey"))
	}

	netKey, err := crypto.GenerateKey()
	if err != nil {
		panic(errors.Wrap(err, "error generating network key"))
	}

	netCfg := &network.RemoteNetworkConfig{
		NotaryGroup:   notaryGroup,
		KeyValueStore: ds,
		SigningKey:    signingKey,
		NetworkKey:    netKey,
	}

	rNet, err := network.NewRemoteNetworkWithConfig(ctx, netCfg)
	if err != nil {
		panic(fmt.Errorf("error setting up network: %v", err))
	}

	net := devnet.DevRemoteNetwork{RemoteNetwork: rNet.(*network.RemoteNetwork)}

	devInkSource, err := net.GetChainTreeByName("dev-ink-source")
	if err != nil {
		panic(fmt.Errorf("error getting dev-ink-source chaintree: %v", err))
	}

	if devInkSource == nil {
		devInkSource, err = net.CreateNamedChainTree("dev-ink-source")
		if err != nil {
			panic(fmt.Errorf("error creating dev-ink-source chaintree: %v", err))
		}
	}

	devInkDID := devInkSource.MustId()

	fmt.Printf("INK_DID=%s\n", devInkDID)

	devInkSourceTree, err := devInkSource.ChainTree.Tree(ctx)
	if err != nil {
		panic(fmt.Errorf("error getting dev-ink-source tree: %v", err))
	}

	devInkTokenName := &consensus.TokenName{ChainTreeDID: devInkDID, LocalName: "ink"}

	devInkLedger := consensus.NewTreeLedger(devInkSourceTree, devInkTokenName)

	devInkExists, err := devInkLedger.TokenExists()
	if err != nil {
		panic(fmt.Errorf("error checking for existence of dev ink: %v", err))
	}

	if !devInkExists {
		fmt.Println("establishing new dev ink token")

		establishInk, err := chaintree.NewEstablishTokenTransaction("ink", 0)
		if err != nil {
			panic(fmt.Errorf("error creating establish_token transaction for dev ink: %v", err))
		}

		devInkSource, _, err = net.PlayTransactions(devInkSource, []*transactions.Transaction{establishInk})
		if err != nil {
			panic(fmt.Errorf("error establishing dev ink token: %v", err))
		}

		devInkSourceTree, err = devInkSource.ChainTree.Tree(ctx)
		if err != nil {
			panic(fmt.Errorf("error getting dev-ink-source tree: %v", err))
		}

		devInkLedger = consensus.NewTreeLedger(devInkSourceTree, devInkTokenName)
	}

	devInkBalance, err := devInkLedger.Balance()
	if err != nil {
		panic(fmt.Errorf("error getting dev ink balance: %v", err))
	}

	fmt.Println("dev ink balance:", devInkBalance)

	if devInkBalance < amount {
		fmt.Println("devink is low; minting some more")

		mintInk, err := chaintree.NewMintTokenTransaction("ink", amount)
		if err != nil {
			panic(fmt.Errorf("error creating mint_token transaction for dev ink: %v", err))
		}

		devInkSource, _, err = net.PlayTransactions(devInkSource, []*transactions.Transaction{mintInk})
		if err != nil {
			panic(fmt.Errorf("error minting dev ink token: %v", err))
		}
	}

	if len(os.Args) > 1 && len(os.Args[1]) > 0 {
		destinationChainId := os.Args[1]

		tokenName, err := consensus.CanonicalTokenName(devInkSource.ChainTree.Dag, devInkDID, "ink", true)
		if err != nil {
			panic(fmt.Errorf("error getting canonical token name for dev ink: %v", err))
		}

		sendTxId := inkSendTxId()

		sendInk, err := chaintree.NewSendTokenTransaction(sendTxId, tokenName.String(), amount, destinationChainId)
		if err != nil {
			panic(fmt.Errorf("error creating send_token transaction for dev ink to %s: %v", destinationChainId, err))
		}

		devInkSource, txResp, err := net.PlayTransactions(devInkSource, []*transactions.Transaction{sendInk})
		if err != nil {
			panic(fmt.Errorf("error sending dev ink token to %s: %v", destinationChainId, err))
		}

		tokenSend, err := consensus.TokenPayloadForTransaction(devInkSource.ChainTree, tokenName, sendTxId, &txResp.Signature)
		if err != nil {
			panic(fmt.Errorf("error generating dev ink token payload: %v", err))
		}

		serializedTokenSend, err := proto.Marshal(tokenSend)
		if err != nil {
			panic(fmt.Errorf("error serializing dev ink token payload: %v", err))
		}

		encodedTokenSend := base64.StdEncoding.EncodeToString(serializedTokenSend)

		fmt.Printf("TOKEN_PAYLOAD=%s\n", encodedTokenSend)
	}
}
