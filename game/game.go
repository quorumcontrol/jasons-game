package game

import (
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
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

type Game struct {
	UI          *ui.JasonsGameUI
	initialTree *chaintree.ChainTree
	cursor      *navigator.Cursor
}

func New(initialTree *chaintree.ChainTree) *Game {
	g := &Game{
		initialTree: initialTree,
	}

	cursor := new(navigator.Cursor).SetChainTree(initialTree)
	g.cursor = cursor

	ui, err := ui.New()
	if err != nil {
		panic(fmt.Errorf("error generating ui: %v", err))
	}

	g.UI = ui
	l, err := g.cursor.GetLocation()
	if err != nil {
		panic(fmt.Errorf("error getting initial location: %v", err))
	}
	g.UI.Write(l.Description)

	ui.EventStream.Subscribe(g.subscriber)

	return g
}

func (g *Game) subscriber(evt interface{}) {
	switch msg := evt.(type) {
	case ui.EventCommand:
		g.handleEventCommand(msg)
	}
}

func (g *Game) handleEventCommand(msg ui.EventCommand) {
	switch msg {
	case "north":
		g.cursor.North()
	case "east":
		g.cursor.East()
	case "south":
		g.cursor.South()
	case "west":
		g.cursor.West()
	}
	l, _ := g.cursor.GetLocation()
	g.UI.Write(l.Description)
}
