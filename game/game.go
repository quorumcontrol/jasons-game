package game

import (
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var log = logging.Logger("game")

const shoutChannel = "jasons-game-shouting-players"

type ping struct{}

type Game struct {
	ui              *actor.PID
	network         network.Network
	player          *Player
	cursor          *navigator.Cursor
	commands        commandList
	messageSequence uint64
	chatSubscriber  *actor.PID
	shoutSubscriber *actor.PID
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
	case *jasonsgame.UserInput:
		g.handleUserInput(actorCtx, msg)
	case *ChatMessage, *ShoutMessage, *JoinMessage:
		g.sendUIMessage(actorCtx, msg)
	case *ping:
		actorCtx.Respond(true)
	}
}

func (g *Game) initialize(actorCtx actor.Context) {
	actorCtx.Send(g.ui, &ui.SetGame{Game: actorCtx.Self()})
	g.shoutSubscriber = actorCtx.Spawn(g.network.PubSubSystem().NewSubscriberProps(shoutChannel))

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

	time.AfterFunc(2*time.Second, func() {
		g.network.PubSubSystem().Broadcast(shoutChannel, &JoinMessage{From: g.player.tree.MustId()})
	})

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

	g.sendUIMessage(
		actorCtx,
		fmt.Sprintf("Created Player %s \n( %s )\nHome: %s \n( %s )",
			playerTree.MustId(),
			playerTree.Tip().String(),
			homeTree.MustId(),
			homeTree.Tip().String()),
	)

	// g.sendUIMessage(actorCtx, "waiting to join the game!")

	l, err := g.cursor.GetLocation()
	if err != nil {
		panic(fmt.Errorf("error getting initial location: %v", err))
	}
	g.sendUIMessage(actorCtx, l)
}

func (g *Game) handleUserInput(actorCtx actor.Context, input *jasonsgame.UserInput) {
	if sender := actorCtx.Sender(); sender != nil {
		log.Debugf("responding to parent with CommandReceived")
		actorCtx.Respond(&jasonsgame.CommandReceived{Sequence: g.messageSequence})
		g.messageSequence++
	}

	cmd, args := g.commands.findCommand(input.Message)
	if cmd != nil {
		log.Debugf("received command %v", cmd.name)
		switch cmd.name {
		case "exit":
			g.sendUIMessage(actorCtx, "exit is unsupported in the browser")
		case "north", "east", "south", "west":
			g.handleLocationInput(actorCtx, cmd, args)
		case "set-description":
			err := g.handleSetDescription(actorCtx, args)
			if err != nil {
				g.sendUIMessage(actorCtx, fmt.Sprintf("error setting description: %v", err))
			}
		case "tip-zoom":
			g.handleTipZoom(actorCtx, args)
		case "say":
			l, err := g.cursor.GetLocation()
			if err != nil {
				panic(errors.Wrap(err, "error getting current location"))
			}
			log.Debugf("publishing chat message")
			g.network.PubSubSystem().Broadcast(topicFromDid(l.Did), &ChatMessage{Message: args})
		case "shout":
			g.network.PubSubSystem().Broadcast(shoutChannel, &ShoutMessage{Message: args})
		default:
			log.Error("unhandled but matched command", cmd.name)
		}
		return
	}
	g.sendUIMessage(actorCtx, "I'm sorry I don't understand.")
}

func (g *Game) handleTipZoom(actorCtx actor.Context, tip string) {
	tipCid, err := cid.Parse(tip)
	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("error parsing tip (%s): %v", tip, err))
		return
	}
	tree, err := g.network.GetTreeByTip(tipCid)
	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("error getting tip: %v", err))
		return
	}

	g.cursor.SetChainTree(tree).SetLocation(0, 0)

	l, err := g.cursor.GetLocation()
	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened: %v", "set-description", err))
	}
	g.sendUIMessage(actorCtx, l)
}

func (g *Game) handleSetDescription(actorCtx actor.Context, desc string) error {
	log.Info("set description")

	tree, err := g.network.GetChainTreeByName("home")
	if err != nil {
		return errors.Wrap(err, "error getting tree by name")
	}

	log.Infof("updating chain %d,%d to %s", g.cursor.X(), g.cursor.Y(), desc)

	updated, err := g.network.UpdateChainTree(tree, fmt.Sprintf("jasons-game/%d/%d", g.cursor.X(), g.cursor.Y()), &jasonsgame.Location{
		Description: desc,
	})

	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened: %v", "set-description", err))
	}

	if g.cursor.Did() == tree.MustId() {
		g.cursor.SetChainTree(updated)
	} else {
		log.Errorf("chain did was not the same %s %s", g.cursor.Did(), tree.MustId())
	}

	log.Info("getting cursor location")
	l, err := g.cursor.GetLocation()
	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened: %v", "set-description", err))
	}
	log.Infof("sending location %v", l)
	g.sendUIMessage(actorCtx, l)

	return err
}

func (g *Game) handleLocationInput(actorCtx actor.Context, cmd *command, args string) {
	oldDid := g.cursor.Did()
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
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened: %v", cmd.name, err))
	}
	if newDid := g.cursor.Did(); newDid != oldDid {
		g.network.StopDiscovery(oldDid)
		g.network.StartDiscovery(newDid)
		if g.chatSubscriber != nil {
			actorCtx.Stop(g.chatSubscriber)
		}
		g.chatSubscriber = actorCtx.Spawn(g.network.PubSubSystem().NewSubscriberProps(topicFromDid(newDid)))
	}

	g.sendUIMessage(actorCtx, l)
}

func topicFromDid(did string) string {
	return fmt.Sprintf("jasons-game-%s", did)
}

func (g *Game) sendUIMessage(actorCtx actor.Context, mesgInter interface{}) {
	msgToUser := &jasonsgame.MessageToUser{
		Sequence: g.messageSequence,
	}
	switch msg := mesgInter.(type) {
	case string:
		msgToUser.Message = msg
	case *jasonsgame.Location:
		msgToUser.Location = msg
		msgToUser.Message = msg.Description
	case *ChatMessage:
		msgToUser.Message = fmt.Sprintf("Someone here says: %s", msg.Message)
	case *ShoutMessage:
		msgToUser.Message = fmt.Sprintf("Someone SHOUTED: %s", msg.Message)
	case *JoinMessage:
		msgToUser.Message = fmt.Sprintf("a new player joined: %s", msg.From)
	default:
		log.Errorf("error, unknown message type: %v", msg)
	}
	actorCtx.Send(g.ui, msgToUser)
	g.messageSequence++
}
