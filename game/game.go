package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/ui"
)

type Game struct {
	ui          *actor.PID
	initialTree *chaintree.ChainTree
	cursor      *navigator.Cursor
}

func NewGameProps(ui *actor.PID, initialTree *chaintree.ChainTree) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Game{
			ui:          ui,
			initialTree: initialTree,
		}
	})
}

func (g *Game) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		g.initialize(actorCtx)
	case *ui.UserInput:
		g.handleUserInput(actorCtx, msg)
	}
}

func (g *Game) initialize(actorCtx actor.Context) {
	cursor := new(navigator.Cursor).SetChainTree(g.initialTree)
	g.cursor = cursor
	actorCtx.Request(g.ui, &ui.Subscribe{})

	l, err := g.cursor.GetLocation()
	if err != nil {
		panic(fmt.Errorf("error getting initial location: %v", err))
	}
	actorCtx.Send(g.ui, l)
}

func (g *Game) handleUserInput(actorCtx actor.Context, input *ui.UserInput) {
	switch input.Message {
	case "north":
		g.cursor.North()
	case "east":
		g.cursor.East()
	case "south":
		g.cursor.South()
	case "west":
		g.cursor.West()
	default:
		actorCtx.Send(g.ui, &ui.MessageToUser{Message: "I'm sorry I don't understand."})
		return
	}
	l, err := g.cursor.GetLocation()
	if err != nil {
		actorCtx.Send(g.ui, &ui.MessageToUser{Message: fmt.Sprintf("some sort of error happened: %v", err)})
	}
	actorCtx.Send(g.ui, l)
}
