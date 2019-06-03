package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var log = logging.Logger("game")

var shoutChannel = []byte("jasons-game-shouting-players")

type ping struct{}

type Game struct {
	ui              *actor.PID
	network         network.Network
	playerTree      *PlayerTree
	commands        commandList
	messageSequence uint64
	locationDid     string
	locationActor   *actor.PID
	chatActor       *actor.PID
	inventoryActor  *actor.PID
}

func NewGameProps(playerTree *PlayerTree, ui *actor.PID, network network.Network) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Game{
			ui:         ui,
			network:    network,
			commands:   defaultCommandList,
			playerTree: playerTree,
		}
	})
}

func (g *Game) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		g.initialize(actorCtx)
	case *jasonsgame.UserInput:
		g.handleUserInput(actorCtx, msg)
	case *jasonsgame.ChatMessage, *jasonsgame.ShoutMessage:
		g.sendUIMessage(actorCtx, msg)
	case *jasonsgame.OpenPortalMessage:
		log.Debugf("received OpenPortalMessage")
		if err := g.handleOpenPortalMessage(actorCtx, msg); err != nil {
			panic(err)
		}
	case *jasonsgame.OpenPortalResponseMessage:
		log.Debugf("received OpenPortalResponseMessage")
		g.handleOpenPortalResponseMessage(actorCtx, msg)
	case *ping:
		actorCtx.Respond(true)
	default:
		log.Warningf("received message of unrecognized type")
	}
}

func (g *Game) initialize(actorCtx actor.Context) {
	actorCtx.Send(g.ui, &ui.SetGame{Game: actorCtx.Self()})
	g.network.Community().SubscribeActor(actorCtx.Self(), shoutChannel)
	_, err := g.network.Community().SubscribeActor(actorCtx.Self(), shoutChannel)
	if err != nil {
		panic(fmt.Errorf("error spawning shout actor: %v", err))
	}

	g.inventoryActor = actorCtx.Spawn(NewInventoryActorProps(&InventoryActorConfig{
		Did:     g.playerTree.Did(),
		Network: g.network,
	}))

	g.setLocation(actorCtx, g.playerTree.HomeLocation.MustId())

	g.sendUIMessage(
		actorCtx,
		fmt.Sprintf("Created Player %s \n( %s )\nHome: %s \n( %s )",
			g.playerTree.Did(),
			g.playerTree.Tip().String(),
			g.playerTree.HomeLocation.MustId(),
			g.playerTree.HomeLocation.Tip().String()),
	)

	l, err := g.getCurrentLocation(actorCtx)

	if err != nil {
		panic(err)
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
		actorCtx.Send(g.chatActor, args)
	case "shout":
		if err := g.network.Community().Send(shoutChannel, &jasonsgame.ShoutMessage{Message: args}); err != nil {
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
	// if did == "" {
	// 	g.sendUIMessage(actorCtx, "you must specify a destination")
	// 	return nil
	// }
	// tree := g.cursor.Tree()
	// loc, err := g.cursor.GetLocation()
	// if err != nil {
	// 	return errors.Wrap(err, "error getting location")
	// }
	// loc.Portal = &jasonsgame.Portal{
	// 	To: did,
	// }

	// err = g.updateLocation(actorCtx, tree, loc)
	// if err != nil {
	// 	return errors.Wrap(err, "error updating location")
	// }

	// g.sendUIMessage(actorCtx, fmt.Sprintf("successfully built a portal to %s", did))
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
	// log.Info("go through portal")
	// l, err := g.cursor.GetLocation()
	// if err != nil {
	// 	return errors.Wrap(err, "error getting location")
	// }
	// if l.Portal == nil {
	// 	return fmt.Errorf("there is no portal where you are")
	// }
	// tree, err := g.network.GetTree(l.Portal.To)
	// if err != nil {
	// 	return errors.Wrap(err, "error getting remote tree")
	// }
	// return g.goToTree(actorCtx, tree)
	return nil
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
	// tree, err := g.network.GetTree(g.cursor.Did())
	// if err != nil {
	// 	return nil, errors.Wrap(err, "error getting remote tree")
	// }
	// g.cursor.SetChainTree(tree)
	// l, err := g.cursor.GetLocation()
	// if err != nil {
	// 	return nil, errors.Wrap(err, fmt.Sprintf("error getting location (%s)", tree.MustId()))
	// }
	// return l, nil
	return nil, nil
}

func (g *Game) goToTree(actorCtx actor.Context, tree *consensus.SignedChainTree) error {
	// oldDid := g.cursor.Did()

	// if newDid := tree.MustId(); newDid != oldDid {
	// 	log.Debugf("moving to a new did %s", newDid)
	// 	g.network.StopDiscovery(oldDid)
	// 	go func() {
	// 		if err := g.network.StartDiscovery(newDid); err != nil {
	// 			log.Errorf("network.StartDiscovery failed: %s", err)
	// 		}
	// 	}()

	// 	// if g.chatSubscriber != nil {
	// 	// 	actorCtx.Stop(g.chatSubscriber)
	// 	// }
	// 	// chatTopic := topicFor(newDid + "-chat")
	// 	// log.Debugf("subscribing to chat at %s", string(chatTopic))
	// 	// g.chatSubscriber = actorCtx.Spawn(g.network.Community().NewSubscriberProps(chatTopic))
	// }

	// g.cursor.SetChainTree(tree).SetLocation(0, 0)

	// l, err := g.cursor.GetLocation()
	// if err != nil {
	// 	return errors.Wrap(err, fmt.Sprintf("error getting location (%s)", tree.MustId()))
	// }
	// g.sendUIMessage(actorCtx, l)
	return nil
}

func (g *Game) handleSetDescription(actorCtx actor.Context, desc string) error {
	response, err := actorCtx.RequestFuture(g.locationActor, &SetLocationDescriptionRequest{Description: desc}, 5*time.Second).Result()

	if err != nil {
		return fmt.Errorf("error setting description: %v", err)
	}

	descriptionResponse, ok := response.(*SetLocationDescriptionResponse)

	if !ok || descriptionResponse.Error != nil {
		return fmt.Errorf("error setting description %v", descriptionResponse.Error)
	}

	g.sendUILocation(actorCtx)
	return nil
}

func (g *Game) handleLocationInput(actorCtx actor.Context, cmd *command, args string) {
	response, err := actorCtx.RequestFuture(g.locationActor, &GetInteraction{Command: cmd.name}, 5*time.Second).Result()

	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened: %v", cmd.name, err))
		return
	}

	interaction, ok := response.(*Interaction)

	// empty location
	if interaction == nil {
		g.goToBarrenWasteland(actorCtx, cmd.name)
		g.sendUILocation(actorCtx)
		return
	}

	if !ok {
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened", cmd.name))
		return
	}

	switch interaction.Action {
	case "respond":
		g.sendUIMessage(actorCtx, interaction.Args["response"])
	case "changeLocation":
		newDid, ok := interaction.Args["did"]

		if !ok {
			g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened %v", cmd.name, "did not found"))
			return
		}

		g.setLocation(actorCtx, newDid)
		g.sendUILocation(actorCtx)
	default:
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened %v", cmd.name, "action not found"))
	}
}

func (g *Game) handleOpenPortal(actorCtx actor.Context, cmd *command, args string) error {
	// splitArgs := []string{}
	// for _, a := range strings.Split(args, " ") {
	// 	if len(strings.TrimSpace(a)) > 0 {
	// 		splitArgs = append(splitArgs, a)
	// 	}
	// }
	// if len(splitArgs) != 2 {
	// 	log.Debugf("received wrong number of arguments (%d): %v", len(splitArgs), splitArgs)
	// 	g.sendUIMessage(actorCtx, "2 arguments required")
	// 	return nil
	// }

	// onLandId := splitArgs[0]
	// loc := splitArgs[1]
	// log.Debugf("requesting to open portal in land ID %q, location %q", onLandId, loc)
	// locArr := strings.Split(loc, ",")
	// if len(locArr) != 2 {
	// 	g.sendUIMessage(actorCtx, "You must specify the location as x,y")
	// 	return nil
	// }
	// x, err := strconv.Atoi(locArr[0])
	// if err != nil {
	// 	g.sendUIMessage(actorCtx, "X coordinate must be numeric")
	// 	return nil
	// }
	// y, err := strconv.Atoi(locArr[1])
	// if err != nil {
	// 	g.sendUIMessage(actorCtx, "Y coordinate must be numeric")
	// 	return nil
	// }

	// playerId := g.playerTree.Did()

	// onLandTree := g.cursor.Tree()
	// toLandId, err := onLandTree.Id()

	// if err != nil {
	// 	return err
	// }

	// log.Debugf("broadcasting OpenPortalMessage, on land ID %s, location (%d, %d), to land ID %s",
	// 	onLandId, x, y, toLandId)
	// if err := g.network.Community().Send(topicFor(onLandId), &jasonsgame.OpenPortalMessage{
	// 	From:      playerId,
	// 	To:        onLandId,
	// 	ToLandId:  toLandId,
	// 	LocationX: int64(x),
	// 	LocationY: int64(y),
	// }); err != nil {
	// 	log.Errorf("failed to broadcast OpenPortalMessage: %s", err)
	// 	return err
	// }

	// g.sendUIMessage(actorCtx, fmt.Sprintf("Requested to open portal on land ID %s", onLandId))
	return nil
}

func (g *Game) handleDropObject(actorCtx actor.Context, args string) error {
	if len(args) == 0 {
		return fmt.Errorf("must give an object name to drop")
	}
	objName := args

	response, err := actorCtx.RequestFuture(g.inventoryActor, &TransferObjectRequest{
		Name: objName,
		To:   g.locationDid,
	}, 5*time.Second).Result()

	if err != nil {
		return fmt.Errorf("error executing drop request: %v", err)
	}

	resp, ok := response.(*TransferObjectResponse)
	if !ok {
		return fmt.Errorf("error casting drop object response")
	}

	if resp.Error != nil {
		return resp.Error
	}

	g.sendUIMessage(actorCtx, fmt.Sprintf("%s has been dropped into your current location", objName))
	return nil
}

func (g *Game) handlePickupObject(actorCtx actor.Context, args string) error {
	if len(args) == 0 {
		return fmt.Errorf("must give an object name to pickup")
	}

	objName := args

	response, err := actorCtx.RequestFuture(g.locationActor, &TransferObjectRequest{
		Name: objName,
		To:   g.playerTree.Did(),
	}, 10*time.Second).Result()

	if err != nil {
		return err
	}

	resp, ok := response.(*TransferObjectResponse)
	if !ok {
		return fmt.Errorf("error casting pickup object response")
	}

	if resp.Error != nil {
		return resp.Error
	}

	g.sendUIMessage(actorCtx, fmt.Sprintf("%s has been picked", objName))
	return nil
}

func (g *Game) handleCreateObject(actorCtx actor.Context, args string) error {
	splitArgs := strings.Split(args, " ")
	objName := splitArgs[0]
	response, err := actorCtx.RequestFuture(g.inventoryActor, &CreateObjectRequest{
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
	response, err := actorCtx.RequestFuture(g.inventoryActor, &InventoryListRequest{}, 5*time.Second).Result()
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

	response, err := actorCtx.RequestFuture(g.locationActor, &InventoryListRequest{}, 5*time.Second).Result()
	if err != nil {
		return err
	}
	inventoryList, ok := response.(*InventoryListResponse)
	if !ok {
		return fmt.Errorf("error casting InventoryListResponse")
	}

	sawSomething := false

	if len(inventoryList.Objects) > 0 {
		sawSomething = true
		g.sendUIMessage(actorCtx, "you see the following objects around you:")
		for objName, obj := range inventoryList.Objects {
			g.sendUIMessage(actorCtx, fmt.Sprintf("%s (%s)", objName, obj.Did))
		}
	}

	// if l.Portal != nil {
	// 	sawSomething = true
	// 	g.sendUIMessage(actorCtx, fmt.Sprintf("you see a mysterious portal leading to %s", l.Portal.To))
	// }

	if !sawSomething {
		g.sendUIMessage(actorCtx, "you look around but don't see anything")
	}

	return nil
}

func topicFor(str string) []byte {
	return []byte(str)
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
	case *jasonsgame.ChatMessage:
		msgToUser.Message = fmt.Sprintf("Someone here says: %s", msg.Message)
	case *jasonsgame.ShoutMessage:
		msgToUser.Message = fmt.Sprintf("Someone SHOUTED: %s", msg.Message)
	default:
		log.Errorf("error, unknown message type: %v", msg)
	}
	actorCtx.Send(g.ui, msgToUser)
	g.messageSequence++
}

func (g *Game) handleOpenPortalMessage(actorCtx actor.Context, msg *jasonsgame.OpenPortalMessage) error {
	// landTree := g.cursor.Tree()
	// landId, err := landTree.Id()
	// if err != nil {
	// 	return err
	// }

	// log.Debugf("handling OpenPortalMessage from %s, location: (%d, %d)", msg.From, msg.LocationX,
	// 	msg.LocationY)
	// g.sendUIMessage(actorCtx, fmt.Sprintf("Player %s wants to open a portal in your land",
	// 	msg.From))
	// g.sendUIMessage(actorCtx, "Request to open portal in your land auto-accepted "+
	// 	"(this is an ALPHA version, future versions will prompt for acceptance)")
	// // TODO: Prompt user for permission

	// log.Debugf("Broadcasting OpenPortalResponseMessage back to sender")
	// if err := g.network.Community().Send(topicFor(msg.From), &jasonsgame.OpenPortalResponseMessage{
	// 	From:      g.playerTree.Did(),
	// 	To:        msg.From,
	// 	Accepted:  true,
	// 	LandId:    landId,
	// 	LocationX: msg.LocationX,
	// 	LocationY: msg.LocationY,
	// }); err != nil {
	// 	return err
	// }

	// loc, err := g.cursor.GetLocation()
	// if err != nil {
	// 	return err
	// }
	// loc.Portal = &jasonsgame.Portal{
	// 	To: msg.ToLandId,
	// }
	// log.Debugf("Playing transaction to add portal to %s at location (%d, %d) in land",
	// 	msg.ToLandId, loc.X, loc.Y)
	// updated, err := g.network.UpdateChainTree(landTree,
	// 	fmt.Sprintf("jasons-game/%d/%d", g.cursor.X(), g.cursor.Y()), loc)
	// if err == nil {
	// 	g.cursor.SetChainTree(updated)
	// }
	return nil
}

func (g *Game) handleOpenPortalResponseMessage(actorCtx actor.Context,
	msg *jasonsgame.OpenPortalResponseMessage) {
	// var uiMsg string
	// if msg.Accepted {
	// 	uiMsg = fmt.Sprintf("Player %s accepted your opening a portal at (%d, %d)", msg.FromPlayer(),
	// 		msg.LocationX, msg.LocationY)
	// } else {
	// 	uiMsg = fmt.Sprintf("Player %s did not accept your opening a portal at (%d, %d)",
	// 		msg.FromPlayer(), msg.LocationX, msg.LocationY)
	// }
	// g.sendUIMessage(actorCtx, uiMsg)
}

func (g *Game) setLocation(actorCtx actor.Context, locationDid string) {
	if g.locationActor != nil {
		actorCtx.Stop(g.locationActor)
	}
	g.locationActor = actorCtx.Spawn(NewLocationActorProps(&LocationActorConfig{
		Network: g.network,
		Did:     locationDid,
	}))
	if g.chatActor != nil {
		actorCtx.Stop(g.chatActor)
	}
	g.chatActor = actorCtx.Spawn(NewChatActorProps(&ChatActorConfig{
		Did:       locationDid,
		Community: g.network.Community(),
	}))
	g.locationDid = locationDid
}

func (g *Game) getCurrentLocation(actorCtx actor.Context) (*jasonsgame.Location, error) {
	response, err := actorCtx.RequestFuture(g.locationActor, &GetLocation{}, 5*time.Second).Result()
	if err != nil {
		return nil, err
	}
	resp, ok := response.(*jasonsgame.Location)
	if !ok {
		return nil, fmt.Errorf("error casting location")
	}
	return resp, nil
}

func (g *Game) goToBarrenWasteland(actorCtx actor.Context, direction string) {
	var inverseLocation string
	switch direction {
	case "north":
		inverseLocation = "south"
	case "south":
		inverseLocation = "north"
	case "east":
		inverseLocation = "west"
	case "west":
		inverseLocation = "east"
	}

	if g.chatActor != nil {
		actorCtx.Stop(g.chatActor)
	}
	if g.locationActor != nil {
		actorCtx.Stop(g.locationActor)
	}

	g.locationActor = actorCtx.Spawn(NewBarrenWastelandLocationActorProps(&BarrenWastelandLocationActorConfig{
		Direction: direction,
		Network:   g.network,
		ReturnInteraction: &Interaction{
			Command: inverseLocation,
			Action:  "changeLocation",
			Args: map[string]string{
				"did": g.locationDid,
			},
		},
		OnExcavate: func(newDid string) {
			g.setLocation(actorCtx, newDid)
			g.sendUILocation(actorCtx)
		},
	}))
	g.locationDid = ""
}

func (g *Game) sendUILocation(actorCtx actor.Context) {
	l, err := g.getCurrentLocation(actorCtx)

	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Errorf("error getting current location: %v", err))
	}

	g.sendUIMessage(actorCtx, l)
}
