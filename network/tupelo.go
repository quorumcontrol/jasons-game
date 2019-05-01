package network

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/tupelo-go-client/client"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
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
		return nil, fmt.Errorf("error creating key: %v", err)
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

	_, err = c.PlayTransactions(tree, ephemeralPrivate, nil, transactions)
	if err != nil {
		return nil, fmt.Errorf("error playing transactions: %v", err)
	}
	return tree, nil

}
