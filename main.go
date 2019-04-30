package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/storage"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
)

var DefaultTree *chaintree.ChainTree

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
	DefaultTree = tree.ChainTree
}

func main() {
	rootCtx := actor.EmptyRootContext
	ui, err := rootCtx.SpawnNamed(ui.NewUIProps(), "ui")
	if err != nil {
		panic(fmt.Errorf("error running UI: %v", err))
	}
	gameActor, err := rootCtx.SpawnNamed(game.NewGameProps(ui, DefaultTree), "game")
	if err != nil {
		panic(fmt.Errorf("error running UI: %v", err))
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()
	fmt.Println("hit ctrl-C one more time to exit")
	<-done
	gameActor.Stop()
}
