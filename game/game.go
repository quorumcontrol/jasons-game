package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
)

var log = logging.Logger("game")

const shoutChannel = "jasons-game-shouting-players"

type ping struct{}

type Game struct {
	ui              *actor.PID
	network         network.Network
	playerTree      *PlayerTree
	cursor          *navigator.Cursor
	commands        commandList
	messageSequence uint64
	chatSubscriber  *actor.PID
	shoutSubscriber *actor.PID
	objectCreator   *actor.PID
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

	var playerChain *consensus.SignedChainTree
	var homeTree *consensus.SignedChainTree

	log.Debug("get player", homeTree)
	playerChain, err := g.network.GetChainTreeByName("player")
	if err != nil {
		log.Error("error getting player: %v", err)
		panic(err)
	}
	if playerChain == nil {
		log.Debug("create player", homeTree)
		playerChain, err = g.network.CreateNamedChainTree("player")
		if err != nil {
			log.Error("error creating player: %v", err)
			panic(err)
		}
		g.playerTree = NewPlayerTree(g.network, playerChain)

		g.playerTree.SetPlayer(&jasonsgame.Player{
			Name: fmt.Sprintf("newb (%s)", playerChain.MustId()),
		})
	} else {
		g.playerTree = NewPlayerTree(g.network, playerChain)
	}

	time.AfterFunc(2*time.Second, func() {
		g.network.PubSubSystem().Broadcast(shoutChannel, &JoinMessage{From: g.playerTree.Did()})
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

	g.chatSubscriber = actorCtx.Spawn(g.network.PubSubSystem().NewSubscriberProps(topicFromDid(homeTree.MustId())))

	cursor := new(navigator.Cursor).SetChainTree(homeTree)
	g.cursor = cursor

	g.objectCreator, err = actorCtx.SpawnNamed(NewCreateObjectActorProps(&CreateObjectActorConfig{
		Player:  g.player,
		Network: g.network,
	}), "objectCreator")
	if err != nil {
		panic(fmt.Errorf("error spawning object creator actor: %v", err))
	}

	g.sendUIMessage(
		actorCtx,
		fmt.Sprintf("Created Player %s \n( %s )\nHome: %s \n( %s )",
			g.playerTree.Did(),
			g.playerTree.Tip().String(),
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
	if cmd == nil {
		g.sendUIMessage(actorCtx, "I'm sorry I don't understand.")
		return
	}

	var err error
	log.Debugf("received command %v", cmd.name)
	switch cmd.name {
	case "exit":
		g.sendUIMessage(actorCtx, "exit is unsupported in the browser")
	case "north", "east", "south", "west":
		g.handleLocationInput(actorCtx, cmd, args)
	case "set-description":
		err = g.handleSetDescription(actorCtx, args)
	case "tip-zoom":
		err = g.handleTipZoom(actorCtx, args)
	case "go-portal":
		err = g.handleGoThroughPortal(actorCtx)
	case "build-portal":
		err = g.handleBuildPortal(actorCtx, args)
	case "say":
		l, err := g.cursor.GetLocation()
		if err == nil {
			log.Debugf("publishing chat message (topic %s)", topicFromDid(l.Did))
			g.network.PubSubSystem().Broadcast(topicFromDid(l.Did), &ChatMessage{Message: args})
		}
	case "shout":
		g.network.PubSubSystem().Broadcast(shoutChannel, &ShoutMessage{Message: args})
	case "create-object":
		err = g.handleCreateObject(actorCtx, args)
	case "help":
		g.sendUIMessage(actorCtx, "available commands:")
		for _, c := range g.commands {
			g.sendUIMessage(actorCtx, c.parse)
		}
	case "name":
		err = g.handleName(args)
	default:
		log.Error("unhandled but matched command", cmd.name)
	}
	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("error with your command: %v", err))
	}
}

func (g *Game) handleName(name string) error {
	log.Debugf("handling set name to %s", name)
	return g.playerTree.SetName(name)
}

func (g *Game) handleBuildPortal(actorCtx actor.Context, did string) error {
	tree := g.cursor.Tree()
	loc, err := g.cursor.GetLocation()
	if err != nil {
		return errors.Wrap(err, "error getting location")
	}
	loc.Portal = &jasonsgame.Portal{
		To: did,
	}

	err = g.updateLocation(actorCtx, tree, loc)
	if err != nil {
		return errors.Wrap(err, "error updating location")
	}

	g.sendUIMessage(actorCtx, fmt.Sprintf("successfully built a portal to %s", did))
	return nil
}

func (g *Game) handleTipZoom(actorCtx actor.Context, tip string) error {
	tipCid, err := cid.Parse(tip)
	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("error parsing tip (%s): %v", tip, err))
		return errors.Wrap(err, fmt.Sprintf("error parsing tip (%s)", tip))
	}
	tree, err := g.network.GetTreeByTip(tipCid)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error getting tip (%s)", tip))
	}

	return g.goToTree(actorCtx, tree)
}

func (g *Game) handleGoThroughPortal(actorCtx actor.Context) error {
	log.Info("go through portal")
	l, err := g.cursor.GetLocation()
	if err != nil {
		return errors.Wrap(err, "error getting location")
	}
	if l.Portal == nil {
		return fmt.Errorf("there is no portal where you are")
	}
	tree, err := g.network.GetTree(l.Portal.To)
	if err != nil {
		return errors.Wrap(err, "error getting remote tree")
	}
	return g.goToTree(actorCtx, tree)
}

func (g *Game) goToTree(actorCtx actor.Context, tree *consensus.SignedChainTree) error {
	oldDid := g.cursor.Did()

	if newDid := tree.MustId(); newDid != oldDid {
		log.Debugf("moving to a new did %s", newDid)
		g.network.StopDiscovery(oldDid)
		go g.network.StartDiscovery(newDid)
		if g.chatSubscriber != nil {
			actorCtx.Stop(g.chatSubscriber)
		}
		log.Debugf("subscribing to %s", topicFromDid(newDid))
		g.chatSubscriber = actorCtx.Spawn(g.network.PubSubSystem().NewSubscriberProps(topicFromDid(newDid)))
	}

	g.cursor.SetChainTree(tree).SetLocation(0, 0)

	l, err := g.cursor.GetLocation()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error getting location (%s)", tree.MustId()))
	}
	g.sendUIMessage(actorCtx, l)
	return nil
}

func (g *Game) handleSetDescription(actorCtx actor.Context, desc string) error {
	log.Info("set description")

	tree := g.cursor.Tree()
	loc, err := g.cursor.GetLocation()
	if err != nil {
		return errors.Wrap(err, "error getting location")
	}

	loc.Description = desc

	log.Infof("updating chain %d,%d to %s", g.cursor.X(), g.cursor.Y(), desc)

	err = g.updateLocation(actorCtx, tree, loc)
	if err != nil {
		return errors.Wrap(err, "error updating location")
	}

	l, err := g.cursor.GetLocation()
	if err != nil {
		return errors.Wrap(err, "error getting location")
	}
	g.sendUIMessage(actorCtx, l)
	return nil
}

func (g *Game) updateLocation(actorCtx actor.Context, tree *consensus.SignedChainTree, location *jasonsgame.Location) error {
	updated, err := g.network.UpdateChainTree(tree, fmt.Sprintf("jasons-game/%d/%d", g.cursor.X(), g.cursor.Y()), location)
	if err != nil {
		return errors.Wrap(err, "error updating chaintree")
	}

	g.cursor.SetChainTree(updated)

	log.Debug("getting cursor location")
	return nil
}

func (g *Game) handleLocationInput(actorCtx actor.Context, cmd *command, args string) {
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

	g.sendUIMessage(actorCtx, l)
}

func (g *Game) handleCreateObject(actorCtx actor.Context, args string) error {
	splitArgs := strings.Split(args, " ")
	objName := splitArgs[0]
	response, err := actorCtx.RequestFuture(g.objectCreator, &CreateObjectRequest{
		Name:        objName,
		Description: strings.Join(splitArgs[1:], " "),
	}, 1*time.Second).Result()
	if err != nil {
		return err
	}

	newObject, ok := response.(*CreateObjectResponse)
	if !ok {
		return fmt.Errorf("error casting create object response")
	}

	g.sendUIMessage(actorCtx, fmt.Sprintf("%s has been created with DID %s and is in your bag of hodling", objName, newObject.Object.ChainTreeDID))
	return nil
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
