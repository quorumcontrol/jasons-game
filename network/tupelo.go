package network

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
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
