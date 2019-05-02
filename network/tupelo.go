package network

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/tupelo-go-sdk/client"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
)

type Tupelo struct {
	key          *ecdsa.PrivateKey
	Store        nodestore.NodeStore
	NotaryGroup  *types.NotaryGroup
	PubSubSystem remote.PubSub
}

func (t *Tupelo) GetTip(did string) (cid.Cid, error) {
	cli := client.New(t.NotaryGroup, did, t.PubSubSystem)
	cli.Listen()
	defer cli.Stop()

	currState, err := cli.TipRequest()
	if err != nil {
		return cid.Undef, errors.Wrap(err, "error getting tip")
	}

	tip, err := cid.Cast(currState.Signature.NewTip)
	if err != nil {
		return cid.Undef, errors.Wrap(err, "error casting tip")
	}
	return tip, nil
}

func (t *Tupelo) CreateChainTree() (*consensus.SignedChainTree, error) {
	ephemeralPrivate, err := crypto.GenerateKey()
	if err != nil {
		return nil, errors.Wrap(err, "error creating key")
	}

	transactions := []*chaintree.Transaction{
		{
			Type: consensus.TransactionTypeSetOwnership,
			Payload: &consensus.SetOwnershipPayload{
				Authentication: []string{
					crypto.PubkeyToAddress(t.key.PublicKey).String(),
				},
			},
		},
	}
	tree, err := consensus.NewSignedChainTree(ephemeralPrivate.PublicKey, t.Store)

	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	_, err = c.PlayTransactions(tree, ephemeralPrivate, nil, transactions)
	if err != nil {
		return nil, errors.Wrap(err, "error playing transactions")
	}
	return tree, nil
}

func (t *Tupelo) UpdateChainTree(tree *consensus.SignedChainTree, path string, value interface{}) error {
	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	log.Debug("UpdateChainTree", "path", path, "value", value)

	transactions := []*chaintree.Transaction{
		{
			Type: consensus.TransactionTypeSetData,
			Payload: &consensus.SetDataPayload{
				Path:  path,
				Value: value,
			},
		},
	}

	var tipPtr *cid.Cid
	if !tree.IsGenesis() {
		tip := tree.Tip()
		tipPtr = &tip
	}

	_, err := c.PlayTransactions(tree, t.key, tipPtr, transactions)
	return err
}
