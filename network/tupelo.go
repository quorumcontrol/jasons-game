package network

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/client"

	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
)

type Tupelo struct {
	Store        nodestore.DagStore
	NotaryGroup  *types.NotaryGroup
	PubSubSystem remote.PubSub
}

func (t *Tupelo) SubscribeToCurrentStateChanges(did string, fn func(msg *signatures.CurrentState)) (func(), error) {
	cli := client.New(t.NotaryGroup, did, t.PubSubSystem)
	cli.Listen()
	sub, err := cli.SubscribeAll(fn)

	if err != nil {
		return nil, err
	}

	return func() {
		cli.Unsubscribe(sub)
		cli.Stop()
	}, nil
}

func (t *Tupelo) GetTip(did string) (cid.Cid, error) {
	cli := client.New(t.NotaryGroup, did, t.PubSubSystem)
	cli.Listen()
	defer cli.Stop()

	currState, err := cli.TipRequest()
	if err != nil {
		return cid.Undef, errors.Wrap(err, "error getting tip")
	}
	if currState.Signature == nil {
		return cid.Undef, nil
	}

	tip, err := cid.Cast(currState.Signature.NewTip)
	if err != nil {
		return cid.Undef, errors.Wrap(err, "error casting tip")
	}
	return tip, nil
}

func (t *Tupelo) CreateChainTree(key *ecdsa.PrivateKey) (*consensus.SignedChainTree, error) {
	ephemeralPrivate, err := crypto.GenerateKey()
	if err != nil {
		return nil, errors.Wrap(err, "error creating key")
	}

	transaction, err := chaintree.NewSetOwnershipTransaction([]string{crypto.PubkeyToAddress(key.PublicKey).String()})
	if err != nil {
		return nil, errors.Wrap(err, "error creating ownership transaction for chaintree")
	}

	tree, err := consensus.NewSignedChainTree(ephemeralPrivate.PublicKey, t.Store)
	if err != nil {
		return nil, errors.Wrap(err, "error creating new signed chaintree")
	}

	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	_, err = c.PlayTransactions(tree, ephemeralPrivate, nil, []*transactions.Transaction{transaction})
	if err != nil {
		return nil, errors.Wrap(err, "error playing transactions")
	}
	return tree, nil
}

func (t *Tupelo) UpdateChainTree(tree *consensus.SignedChainTree, key *ecdsa.PrivateKey, path string, value interface{}) error {
	log.Debug("UpdateChainTree", "did", tree.MustId(), "path", path, "value", value)

	transaction, err := chaintree.NewSetDataTransaction(path, value)
	if err != nil {
		return errors.Wrap(err, "error creating set data transaction")
	}

	_, err = t.PlayTransactions(tree, key, []*transactions.Transaction{transaction})
	return err
}

func (t *Tupelo) PlayTransactions(tree *consensus.SignedChainTree, key *ecdsa.PrivateKey, transactions []*transactions.Transaction) (*consensus.AddBlockResponse, error) {
	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	var tipPtr *cid.Cid
	if !tree.IsGenesis() {
		tip := tree.Tip()
		tipPtr = &tip
	}

	return c.PlayTransactions(tree, key, tipPtr, transactions)
}

func (t *Tupelo) TokenPayloadForTransaction(tree *consensus.SignedChainTree, tokenName *consensus.TokenName, sendTokenTxId string, sendTxSig *signatures.Signature) (*transactions.TokenPayload, error) {
	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	return c.TokenPayloadForTransaction(tree.ChainTree, tokenName, sendTokenTxId, sendTxSig)
}
