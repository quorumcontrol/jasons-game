package game

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
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
	case "go-portal":
		g.handleInteractionInput(actorCtx, cmd, args)
	case "set-description":
		err = g.handleSetDescription(actorCtx, args)
	case "tip-zoom":
		err = g.handleTipZoom(actorCtx, args)
	case "refresh":
		g.sendUILocation(actorCtx)
	case "build-portal":
		err = g.handleBuildPortal(actorCtx, args)
	case "say":
		actorCtx.Send(g.chatActor, args)
	case "shout":
		if err := g.network.Community().Send(shoutChannel, &jasonsgame.ShoutMessage{Message: args}); err != nil {
			log.Errorf("failed to broadcast ShoutMessage: %s", err)
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
	case "create-location":
		err = g.handleCreateLocation(actorCtx, args)
	case "connect-location":
		err = g.handleConnectLocation(actorCtx, args)
	case "help":
		g.sendUIMessage(actorCtx, "available commands:")
		for _, c := range g.commands {
			g.sendUIMessage(actorCtx, c.parse)
		}
	case "name":
		err = g.handleName(args)
	case "interaction":
		g.handleInteractionInput(actorCtx, cmd, args)
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

func (g *Game) handleBuildPortal(actorCtx actor.Context, toDid string) error {
	response, err := actorCtx.RequestFuture(g.locationActor, &BuildPortalRequest{
		To: toDid,
	}, 5*time.Second).Result()
	if err != nil {
		return fmt.Errorf("Error building portal: %v", err)
	}

	if respErr := response.(*BuildPortalResponse).Error; respErr != nil {
		return fmt.Errorf("Error building portal: %v", respErr)
	}

	g.sendUIMessage(actorCtx, fmt.Sprintf("successfully built a portal to %s", toDid))
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

	g.setLocation(actorCtx, tree.MustId())
	g.sendUILocation(actorCtx)
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

func (g *Game) handleInteractionInput(actorCtx actor.Context, cmd *command, args string) {
	interactionInput := cmd.parse
	response, err := actorCtx.RequestFuture(g.locationActor, &GetInteractionRequest{Command: interactionInput}, 5*time.Second).Result()
	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened: %v", interactionInput, err))
		return
	}

	interaction, ok := response.(*Interaction)

	if interaction == nil {
		g.sendUIMessage(actorCtx, fmt.Sprintf("no interaction matching %s %s", cmd.parse, args))
		return
	}

	if !ok {
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened", interactionInput))
		return
	}

	switch interaction.Action {
	case "respond":
		g.sendUIMessage(actorCtx, interaction.Args["response"])
	case "changeLocation":
		newDid, ok := interaction.Args["did"]

		if !ok {
			g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened %v", interactionInput, "did not found"))
			return
		}

		g.setLocation(actorCtx, newDid)
		g.sendUILocation(actorCtx)
	default:
		g.sendUIMessage(actorCtx, fmt.Sprintf("%s some sort of error happened %v", interactionInput, "action not found"))
	}
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
	response, err := actorCtx.RequestFuture(g.locationActor, &InventoryListRequest{}, 5*time.Second).Result()
	if err != nil {
		return err
	}
	inventoryList, ok := response.(*InventoryListResponse)
	if !ok {
		return fmt.Errorf("error casting InventoryListResponse")
	}

	l, err := g.getCurrentLocation(actorCtx)
	if err != nil {
		return fmt.Errorf("error getting current location: %v", err)
	}

	sawSomething := false

	if len(inventoryList.Objects) > 0 {
		sawSomething = true
		g.sendUIMessage(actorCtx, "you see the following objects around you:")
		for objName, obj := range inventoryList.Objects {
			g.sendUIMessage(actorCtx, fmt.Sprintf("%s (%s)", objName, obj.Did))
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

func (g *Game) handleCreateLocation(actorCtx actor.Context, args string) error {
	newLocation, err := g.network.CreateChainTree()
	if err != nil {
		return err
	}

	g.sendUIMessage(actorCtx, "new location created "+newLocation.MustId())
	return nil
}

func (g *Game) handleConnectLocation(actorCtx actor.Context, args string) error {
	connectRegex := regexp.MustCompile(`^(did:tupelo:\w+) as (.*)`)
	matches := connectRegex.FindStringSubmatch(args)

	if len(matches) < 2 {
		return fmt.Errorf("must specify connections in the syntax of: connect location DID as CMD")
	}

	toDid := matches[1]
	interactionCommand := matches[2]

	targetTree, err := g.network.GetTree(toDid)
	if err != nil {
		return fmt.Errorf("error fetching target location: %v", err)
	}
	if targetTree == nil {
		return fmt.Errorf("could not find target location")
	}

	loc := NewLocationTree(g.network, targetTree)

	keys, err := g.playerTree.Keys()
	if err != nil {
		return fmt.Errorf("error fetching player keys")
	}
	isOwnedBy, _ := loc.IsOwnedBy(keys)
	if !isOwnedBy {
		return fmt.Errorf("can't connect a location that you don't own")
	}

	interaction := &Interaction{
		Command: interactionCommand,
		Action:  "changeLocation",
		Args: map[string]string{
			"did": toDid,
		},
	}

	result, err := actorCtx.RequestFuture(g.locationActor, &AddInteractionRequest{Interaction: interaction}, 5*time.Second).Result()
	if err != nil {
		return fmt.Errorf("error adding connection: %v", err)
	}

	resp, ok := result.(*AddInteractionResponse)
	if !ok {
		return fmt.Errorf("error casting location")
	}
	if resp.Error != nil {
		return fmt.Errorf("error adding connection: %v", resp.Error)
	}

	g.attachInteractions(actorCtx)
	g.sendUIMessage(actorCtx, fmt.Sprintf("added a connection to %s as %s", toDid, interactionCommand))
	return nil
}

func topicFor(str string) []byte {
	return []byte(str)
}

func (g *Game) sendUILocation(actorCtx actor.Context) {
	l, err := g.getCurrentLocation(actorCtx)

	if err != nil {
		g.sendUIMessage(actorCtx, fmt.Errorf("error getting current location: %v", err))
	}

	g.sendUIMessage(actorCtx, l)
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

func (g *Game) setLocation(actorCtx actor.Context, locationDid string) {
	if g.locationActor != nil {
		actorCtx.Stop(g.locationActor)
	}
	g.locationActor = actorCtx.Spawn(NewLocationActorProps(&LocationActorConfig{
		Network: g.network,
		Did:     locationDid,
	}))
	g.locationDid = locationDid

	err := g.attachInteractions(actorCtx)
	if err != nil {
		panic(errors.Wrap(err, "error attaching interactions for location"))
	}

	if g.chatActor != nil {
		actorCtx.Stop(g.chatActor)
	}
	g.chatActor = actorCtx.Spawn(NewChatActorProps(&ChatActorConfig{
		Did:       locationDid,
		Community: g.network.Community(),
	}))
}

func (g *Game) attachInteractions(actorCtx actor.Context) error {
	response, err := actorCtx.RequestFuture(g.locationActor, &ListInteractionsRequest{}, 5*time.Second).Result()
	if err != nil || response == nil {
		return err
	}

	interactionsResponse, ok := response.(*ListInteractionsResponse)
	if !ok {
		return fmt.Errorf("error casting ListInteractionsResponse")
	}
	if interactionsResponse.Error != nil {
		return interactionsResponse.Error
	}

	interactions := interactionsResponse.Interactions
	interactionCommands := make(commandList, len(interactions))
	for i, cmd := range interactions {
		interactionCommands[i] = newCommand("interaction", cmd)
	}

	g.commands = append(defaultCommandList, interactionCommands...)
	return nil
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
