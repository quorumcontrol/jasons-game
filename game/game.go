package game

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/messages"

	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	gossip3messages "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
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
	inventory       *actor.PID
	home            *actor.PID
	broadcaster     *messages.Broadcaster
}

func NewGameProps(playerTree *PlayerTree, ui *actor.PID,
	network network.Network, broadcaster *messages.Broadcaster) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Game{
			ui:          ui,
			network:     network,
			commands:    defaultCommandList,
			broadcaster: broadcaster,
			playerTree:  playerTree,
		}
	})
}

func (g *Game) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		g.initialize(actorCtx)
	case *jasonsgame.UserInput:
		g.handleUserInput(actorCtx, msg)
	case *messages.ChatMessage, *messages.ShoutMessage:
		g.sendUIMessage(actorCtx, msg)
	case *messages.OpenPortalMessage:
		log.Debugf("received OpenPortalMessage")
		if err := g.handleOpenPortalMessage(actorCtx, msg); err != nil {
			panic(err)
		}
	case *messages.OpenPortalResponseMessage:
		log.Debugf("received OpenPortalResponseMessage")
		g.handleOpenPortalResponseMessage(actorCtx, msg)
	case *ping:
		actorCtx.Respond(true)
	case gossip3messages.WireMessage:
		log.Warningf("received message of unrecognized type, typeCode: %d", msg.TypeCode())
	default:
		log.Warningf("received message of unrecognized type")
	}
}

func (g *Game) initialize(actorCtx actor.Context) {
	actorCtx.Send(g.ui, &ui.SetGame{Game: actorCtx.Self()})
	g.shoutSubscriber = actorCtx.Spawn(g.network.PubSubSystem().NewSubscriberProps(shoutChannel))

	landTopic := topicFromDid(g.playerTree.HomeTree.MustId())
	log.Debugf("subscribing to messages with our land as topic %s", landTopic)
	// TODO: Use general, non-specific, pubsub topic instead
	g.chatSubscriber = actorCtx.Spawn(g.network.PubSubSystem().NewSubscriberProps(landTopic))

	cursor := new(navigator.Cursor).SetChainTree(g.playerTree.HomeTree)
	g.cursor = cursor

	g.home, err = actorCtx.SpawnNamed(NewLandActorProps(&LandActorConfig{
		Did:     homeTree.MustId(),
		Network: g.network,
	}), "home")
	if err != nil {
		panic(fmt.Errorf("error spawning home actor: %v", err))
	}

	// TODO: switch to global topic
	var err error
	g.inventory, err = actorCtx.SpawnNamed(NewInventoryActorProps(&InventoryActorConfig{
		Player:  g.playerTree,
		Network: g.network,
	}), "inventory")
	if err != nil {
		panic(fmt.Errorf("error spawning inventory actor: %v", err))
	}

	g.sendUIMessage(
		actorCtx,
		fmt.Sprintf("Created Player %s \n( %s )\nHome: %s \n( %s )",
			g.playerTree.Did(),
			g.playerTree.Tip().String(),
			g.playerTree.HomeTree.MustId(),
			g.playerTree.HomeTree.Tip().String()),
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
	case "refresh":
		err = g.handleRefresh(actorCtx)
	case "build-portal":
		err = g.handleBuildPortal(actorCtx, args)
	case "say":
		l, err := g.cursor.GetLocation()
		if err == nil {
			// TODO: Use general, non-specific, pubsub topic instead, designating recipient through a
			// field.
			chatTopic := topicFromDid(l.Did)
			log.Debugf("publishing chat message (topic %s)", chatTopic)
			if err := g.broadcaster.Broadcast(chatTopic, &messages.ChatMessage{Message: args}); err != nil {
				log.Errorf("failed to broadcast ChatMessage: %s", err)
			}
		}
	case "shout":
		if err := g.broadcaster.Broadcast(shoutChannel, &messages.ShoutMessage{Message: args}); err != nil {
			log.Errorf("failed to broadcast ShoutMessage: %s", err)
		}
	case "open-portal":
		if err := g.handleOpenPortal(actorCtx, cmd, args); err != nil {
			log.Errorf("g.handleOpenPortal failed: %s", err)
		}
	case "create-object":
		err = g.handleCreateObject(actorCtx, args)
	case "drop-object":
		err = g.handleDropObject(actorCtx, args)
	case "pick-up-object":
		err = g.handlePickupObject(actorCtx, args)
	case "player-inventory-list":
		err = g.handlePlayerInventoryList(actorCtx)
	case "location-inventory-list":
		err = g.handleLocationInventoryList(actorCtx)
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

func (g *Game) handleRefresh(actorCtx actor.Context) error {
	l, err := g.refreshLocation()
	if err != nil {
		return err
	}
	g.sendUIMessage(actorCtx, l)
	return nil
}

func (g *Game) refreshLocation() (*jasonsgame.Location, error) {
	tree, err := g.network.GetTree(g.cursor.Did())
	if err != nil {
		return nil, errors.Wrap(err, "error getting remote tree")
	}
	g.cursor.SetChainTree(tree)
	l, err := g.cursor.GetLocation()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error getting location (%s)", tree.MustId()))
	}
	return l, nil
}

func (g *Game) goToTree(actorCtx actor.Context, tree *consensus.SignedChainTree) error {
	oldDid := g.cursor.Did()

	if newDid := tree.MustId(); newDid != oldDid {
		log.Debugf("moving to a new did %s", newDid)
		g.network.StopDiscovery(oldDid)
		go func() {
			if err := g.network.StartDiscovery(newDid); err != nil {
				log.Errorf("network.StartDiscovery failed: %s", err)
			}
		}()
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

func (g *Game) handleOpenPortal(actorCtx actor.Context, cmd *command, args string) error {
	splitArgs := []string{}
	for _, a := range strings.Split(args, " ") {
		if len(strings.TrimSpace(a)) > 0 {
			splitArgs = append(splitArgs, a)
		}
	}
	if len(splitArgs) != 2 {
		log.Debugf("received wrong number of arguments (%d): %v", len(splitArgs), splitArgs)
		g.sendUIMessage(actorCtx, "2 arguments required")
		return nil
	}

	onLandId := splitArgs[0]
	loc := splitArgs[1]
	log.Debugf("requesting to open portal in land ID %q, location %q", onLandId, loc)
	locArr := strings.Split(loc, ",")
	if len(locArr) != 2 {
		g.sendUIMessage(actorCtx, "You must specify the location as x,y")
		return nil
	}
	x, err := strconv.Atoi(locArr[0])
	if err != nil {
		g.sendUIMessage(actorCtx, "X coordinate must be numeric")
		return nil
	}
	y, err := strconv.Atoi(locArr[1])
	if err != nil {
		g.sendUIMessage(actorCtx, "Y coordinate must be numeric")
		return nil
	}

	playerId := g.playerTree.Did()

	onLandTree := g.cursor.Tree()
	toLandId, err := onLandTree.Id()

	if err != nil {
		return err
	}

	log.Debugf("broadcasting OpenPortalMessage, on land ID %s, location (%d, %d), to land ID %s",
		onLandId, x, y, toLandId)
	if err := g.broadcaster.BroadcastGeneral(&messages.OpenPortalMessage{
		From:      playerId,
		To:        onLandId,
		ToLandId:  toLandId,
		LocationX: int64(x),
		LocationY: int64(y),
	}); err != nil {
		log.Errorf("failed to broadcast OpenPortalMessage: %s", err)
		return err
	}

	g.sendUIMessage(actorCtx, fmt.Sprintf("Requested to open portal on land ID %s", onLandId))
	return nil
}

func (g *Game) handleDropObject(actorCtx actor.Context, args string) error {
	if len(args) == 0 {
		g.sendUIMessage(actorCtx, "must give an object name to drop")
		return nil
	}
	objName := args

	g.refreshLocation()
	l, err := g.cursor.GetLocation()
	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("Could not find your current location: %v", err))
		return fmt.Errorf("Error on GetLocation %v", err)
	}

	response, err := actorCtx.RequestFuture(g.inventory, &DropObjectRequest{
		Name:     objName,
		Location: l,
	}, 5*time.Second).Result()

	resp, ok := response.(*DropObjectResponse)
	if !ok {
		return fmt.Errorf("error casting drop object response")
	}

	if resp.Error != nil {
		g.sendUIMessage(actorCtx, resp.Error)
		return resp.Error
	}

	g.refreshLocation()
	g.sendUIMessage(actorCtx, fmt.Sprintf("%s has been dropped into %v", objName, l.PrettyString()))
	return nil
}

func (g *Game) handlePickupObject(actorCtx actor.Context, args string) error {
	if len(args) == 0 {
		g.sendUIMessage(actorCtx, "must give an object name to pickup")
		return nil
	}

	objName := args

	g.refreshLocation()
	l, err := g.cursor.GetLocation()
	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("Could not find your current location: %v", err))
		return fmt.Errorf("Error on GetLocation %v", err)
	}

	response, err := actorCtx.RequestFuture(g.inventory, &PickupObjectRequest{
		Name:     objName,
		Location: l,
	}, 10*time.Second).Result()

	resp, ok := response.(*PickupObjectResponse)
	if !ok {
		return fmt.Errorf("error casting pickup object response")
	}

	if resp.Error != nil {
		g.sendUIMessage(actorCtx, resp.Error)
		return resp.Error
	}

	g.refreshLocation()
	g.sendUIMessage(actorCtx, fmt.Sprintf("%s has been picked up from %v", objName, l.PrettyString()))
	return nil
}

func (g *Game) handleCreateObject(actorCtx actor.Context, args string) error {
	splitArgs := strings.Split(args, " ")
	objName := splitArgs[0]
	response, err := actorCtx.RequestFuture(g.inventory, &CreateObjectRequest{
		Name:        objName,
		Description: strings.Join(splitArgs[1:], " "),
	}, 5*time.Second).Result()
	if err != nil {
		return err
	}

	newObject, ok := response.(*CreateObjectResponse)
	if !ok {
		return fmt.Errorf("error casting create object response")
	}

	g.sendUIMessage(actorCtx, fmt.Sprintf("%s has been created with DID %s and is in your bag of hodling", objName, newObject.Object.Did))
	return nil
}

func (g *Game) handlePlayerInventoryList(actorCtx actor.Context) error {
	response, err := actorCtx.RequestFuture(g.inventory, &InventoryListRequest{}, 5*time.Second).Result()
	if err != nil {
		return err
	}
	inventoryList, ok := response.(*InventoryListResponse)
	if !ok {
		return fmt.Errorf("error casting InventoryListResponse")
	}

	if len(inventoryList.Objects) == 0 {
		g.sendUIMessage(actorCtx, "your bag of hodling appears to be empty")
		return nil
	}

	g.sendUIMessage(actorCtx, "inside of your bag of hodling you find:")
	for objName, obj := range inventoryList.Objects {
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s (%s)", objName, obj.Did))
	}
	return nil
}

func (g *Game) handleLocationInventoryList(actorCtx actor.Context) error {
	g.refreshLocation()
	l, err := g.cursor.GetLocation()
	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("Could not find your current location: %v", err))
		return fmt.Errorf("Error on GetLocation %v", err)
	}

	sawSomething := false

	if len(l.Inventory) > 0 {
		sawSomething = true
		g.sendUIMessage(actorCtx, "you see the following objects around you:")
		for objName, objDid := range l.Inventory {
			g.sendUIMessage(actorCtx, fmt.Sprintf("%s (%s)", objName, objDid))
		}
	}

	if l.Portal != nil {
		sawSomething = true
		g.sendUIMessage(actorCtx, fmt.Sprintf("you see a mysterious portal leading to %s", l.Portal.To))
	}

	if !sawSomething {
		g.sendUIMessage(actorCtx, "you look around but don't see anything")
	}
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
	case *messages.ChatMessage:
		msgToUser.Message = fmt.Sprintf("Someone here says: %s", msg.Message)
	case *messages.ShoutMessage:
		msgToUser.Message = fmt.Sprintf("Someone SHOUTED: %s", msg.Message)
	default:
		log.Errorf("error, unknown message type: %v", msg)
	}
	actorCtx.Send(g.ui, msgToUser)
	g.messageSequence++
}

func (g *Game) handleOpenPortalMessage(actorCtx actor.Context, msg *messages.OpenPortalMessage) error {
	landTree := g.cursor.Tree()
	landId, err := landTree.Id()
	if err != nil {
		return err
	}

	log.Debugf("handling OpenPortalMessage from %s, location: (%d, %d)", msg.From, msg.LocationX,
		msg.LocationY)
	g.sendUIMessage(actorCtx, fmt.Sprintf("Player %s wants to open a portal in your land",
		msg.From))
	g.sendUIMessage(actorCtx, "Request to open portal in your land auto-accepted "+
		"(this is an ALPHA version, future versions will prompt for acceptance)")
	// TODO: Prompt user for permission

	log.Debugf("Broadcasting OpenPortalResponseMessage back to sender")
	if err := g.broadcaster.BroadcastGeneral(&messages.OpenPortalResponseMessage{
		From:      g.playerTree.Did(),
		To:        msg.From,
		Accepted:  true,
		LandId:    landId,
		LocationX: msg.LocationX,
		LocationY: msg.LocationY,
	}); err != nil {
		return err
	}

	loc, err := g.cursor.GetLocation()
	if err != nil {
		return err
	}
	loc.Portal = &jasonsgame.Portal{
		To: msg.ToLandId,
	}
	log.Debugf("Playing transaction to add portal to %s at location (%d, %d) in land",
		msg.ToLandId, loc.X, loc.Y)
	updated, err := g.network.UpdateChainTree(landTree,
		fmt.Sprintf("jasons-game/%d/%d", g.cursor.X(), g.cursor.Y()), loc)
	if err == nil {
		g.cursor.SetChainTree(updated)
	}

	return nil
}

func (g *Game) handleOpenPortalResponseMessage(actorCtx actor.Context,
	msg *messages.OpenPortalResponseMessage) {
	var uiMsg string
	if msg.Accepted {
		uiMsg = fmt.Sprintf("Player %s accepted your opening a portal at (%d, %d)", msg.FromPlayer(),
			msg.LocationX, msg.LocationY)
	} else {
		uiMsg = fmt.Sprintf("Player %s did not accept your opening a portal at (%d, %d)",
			msg.FromPlayer(), msg.LocationX, msg.LocationY)
	}
	g.sendUIMessage(actorCtx, uiMsg)
}
