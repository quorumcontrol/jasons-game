package game

import (
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/storage"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
)

var DefaultTree *consensus.SignedChainTree

func init() {
	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())

	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Errorf("error creating key: %v", err))
	}

	tree, err := consensus.NewSignedChainTree(key.PublicKey, store)
	if err != nil {
		panic(fmt.Errorf("error creating chain: %v", err))
	}
	updated, err := tree.ChainTree.Dag.SetAsLink([]string{"tree", "data", "jasons-game", "0", "0"}, &navigator.Location{Description: "hi, welcome"})
	if err != nil {
		panic(fmt.Errorf("error updating dag: %v", err))
	}
	updated, err = updated.SetAsLink([]string{"tree", "data", "jasons-game", "0", "1"}, &navigator.Location{Description: "you are north of the welcome"})
	if err != nil {
		panic(fmt.Errorf("error updating dag: %v", err))
	}
	tree.ChainTree.Dag = updated
	DefaultTree = tree
}
