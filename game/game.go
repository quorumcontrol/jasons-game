package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
	"github.com/sbstjn/allot"
)

var log = logging.Logger("game")

var defaultCommandList = commandList{
	{name: "north", parse: "north"},
	{name: "south", parse: "south"},
	{name: "west", parse: "west"},
	{name: "east", parse: "east"},
	{name: "name", parse: "call me <name:string>"},
}

type ping struct{}

type Game struct {
	ui       *actor.PID
	network  network.Network
	player   *Player
	cursor   *navigator.Cursor
	commands commandList
}

func NewGameProps(ui *actor.PID, network network.Network) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Game{
			ui:       ui,
			network:  network,
			commands: defaultCommandList,
		}
	})
}

func (g *Game) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		g.initialize(actorCtx)
	case *ui.UserInput:
		g.handleUserInput(actorCtx, msg)
	case *ping:
		actorCtx.Respond(true)
	}
}

func (g *Game) initialize(actorCtx actor.Context) {
	var playerTree *consensus.SignedChainTree
	var homeTree *consensus.SignedChainTree

	log.Debug("get player", homeTree)
	playerTree, err := g.network.GetChainTreeByName("player")
	if err != nil {
		log.Error("error getting player: %v", err)
		panic(err)
	}
	if playerTree == nil {
		log.Debug("create player", homeTree)
		playerTree, err = g.network.CreateNamedChainTree("player")
		if err != nil {
			log.Error("error creating player: %v", err)
			panic(err)
		}
	}
	g.player = NewPlayer(playerTree)

	homeTree, err = g.network.GetChainTreeByName("home")
	log.Debug("get home", homeTree)
	if err != nil {
		panic(err)
	}
	if homeTree == nil {
		log.Debug("create home")
		homeTree, err = createHome(g.network)
		if err != nil {
			log.Error("error creating home", err)
			panic(err)
		}
	}

	cursor := new(navigator.Cursor).SetChainTree(homeTree)
	g.cursor = cursor
	actorCtx.Request(g.ui, &ui.Subscribe{})

	l, err := g.cursor.GetLocation()
	if err != nil {
		panic(fmt.Errorf("error getting initial location: %v", err))
	}
	actorCtx.Send(g.ui, l)
}

func (g *Game) handleUserInput(actorCtx actor.Context, input *ui.UserInput) {
	cmd, matches := g.commands.findCommand(input.Message)
	if cmd != nil {
		switch cmd.name {
		case "exit":
			actorCtx.Send(g.ui, &ui.Exit{})
		case "north", "east", "south", "west":
			g.handleLocationInput(actorCtx, cmd, matches)
		default:
			log.Error("unhandled but matched command", cmd.name)
		}
		return
	}
	actorCtx.Send(g.ui, &ui.MessageToUser{Message: "I'm sorry I don't understand."})
}

func (g *Game) handleLocationInput(actorCtx actor.Context, cmd *command, matches allot.MatchInterface) {
	switch cmd.name {
	case "north":
		g.cursor.North()
	case "east":
		g.cursor.East()
	case "south":
		g.cursor.South()
	case "west":
		g.cursor.West()
	}
	l, err := g.cursor.GetLocation()
	if err != nil {
		actorCtx.Send(g.ui, &ui.MessageToUser{Message: fmt.Sprintf("some sort of error happened: %v", err)})
	}
	actorCtx.Send(g.ui, l)
}
