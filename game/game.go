package game

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	proto "github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
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
	ui                   *actor.PID
	network              network.Network
	playerTree           *PlayerTree
	commands             commandList
	messageSequence      uint64
	locationDid          string
	locationActor        *actor.PID
	chatActor            *actor.PID
	inventoryActor       *actor.PID
	shoutActor           *actor.PID
	commandsByActorCache map[*actor.PID]commandList
}

type StateChange struct {
	PID *actor.PID
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
	case *jasonsgame.ImportRequest:
		g.handleImportRequest(actorCtx, msg)
	case *jasonsgame.CommandUpdate:
		g.sendCommandUpdate(actorCtx)
	case *jasonsgame.ChatMessage, *jasonsgame.ShoutMessage:
		g.sendUserMessage(actorCtx, msg)
	case *StateChange:
		g.handleStateChange(actorCtx, msg)
	case *ping:
		actorCtx.Respond(true)
	default:
		log.Warningf("received unrecognized message %#v", msg)
	}
}

func (g *Game) initialize(actorCtx actor.Context) {
	actorCtx.Send(g.ui, &ui.SetGame{Game: actorCtx.Self()})

	g.shoutActor = actorCtx.Spawn(g.network.Community().NewSubscriberProps(shoutChannel))

	g.inventoryActor = actorCtx.Spawn(NewInventoryActorProps(&InventoryActorConfig{
		Did:     g.playerTree.Did(),
		Network: g.network,
	}))
	err := g.refreshInteractionsFor(actorCtx, g.inventoryActor)
	if err != nil {
		panic(errors.Wrap(err, "error attaching interactions for inventory"))
	}

	g.setLocation(actorCtx, g.playerTree.HomeLocation.MustId())

	g.sendUserMessage(
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

	g.sendUserMessage(actorCtx, l)
}

func (g *Game) handleImportLocation(actorCtx actor.Context, input *jasonsgame.LocationSpec) error {
	newLocation, err := g.createLocation(actorCtx)
	if err != nil {
		return err
	}

	id := newLocation.MustId()

	g.sendUserMessage(actorCtx, fmt.Sprintf("new location %s imported with id %s ", input.Name, id))

	input.Did = id
	g.sendImportResult(actorCtx, input)

	return nil
}

func (g *Game) handlePopulateLocation(actorCtx actor.Context, input *jasonsgame.PopulateSpec) error {
	var err error
	switch phase := input.GetPhase().(type) {
	case *jasonsgame.PopulateSpec_Visit:
		g.sendUserMessage(actorCtx, fmt.Sprintf("visiting location %s (id: %s)", input.Name, input.Did))
		err = g.handleLocationZoom(actorCtx, input.Did)
	case *jasonsgame.PopulateSpec_Describe:
		g.sendUserMessage(actorCtx, fmt.Sprintf("setting description for location %s to '%s'", input.Name, phase.Describe.Description))
		err = g.handleSetDescription(actorCtx, phase.Describe.Description)
	case *jasonsgame.PopulateSpec_Drop:
		if phase.Drop.ObjectDid != "" {
			interaction := &DropObjectInteraction{Did: phase.Drop.ObjectDid}
			g.sendUserMessage(actorCtx, fmt.Sprintf("dropping object %s at location %s", phase.Drop.ObjectName, input.Name))
			err = g.handleDropObject(actorCtx, interaction)
		} else {
			g.sendUserMessage(actorCtx, "No objects to drop")
		}
	case *jasonsgame.PopulateSpec_Connect:
		g.sendUserMessage(actorCtx, fmt.Sprintf("connecting locations %s and %s", input.Name, phase.Connect.ConnectionName))
		err = g.connectLocations(actorCtx, phase.Connect.ToDid, phase.Connect.ConnectionName)
	}

	if err != nil {
		g.sendUserMessage(actorCtx, fmt.Sprintf("error importing: %v", err))
		return err
	}

	g.sendImportResult(actorCtx, input)
	return nil
}

func (g *Game) handleImportObject(actorCtx actor.Context, input *jasonsgame.ObjectSpec) error {
	newObject, err := g.createObject(actorCtx, input.Name, input.Description)
	if err != nil {
		return err
	}

	did := newObject.Object.Did

	g.sendUserMessage(actorCtx, fmt.Sprintf("new object %s imported with DID %s", input.Name, did))

	input.Did = did
	g.sendImportResult(actorCtx, input)

	return nil
}

func (g *Game) handleImportRequest(actorCtx actor.Context, input *jasonsgame.ImportRequest) {
	if sender := actorCtx.Sender(); sender != nil {
		log.Debugf("responding to parent with CommandReceived")
		actorCtx.Respond(&jasonsgame.CommandReceived{Sequence: g.messageSequence})
		g.messageSequence++
	}

	var err error

	spec := input.GetSpec()

	switch input.GetSpec().(type) {
	case *jasonsgame.ImportRequest_Location:
		err = g.handleImportLocation(actorCtx, input.GetLocation())
	case *jasonsgame.ImportRequest_Object:
		err = g.handleImportObject(actorCtx, input.GetObject())
	case *jasonsgame.ImportRequest_Populate:
		err = g.handlePopulateLocation(actorCtx, input.GetPopulate())
	default:
		log.Error("unrecognized input spec", spec)
	}
	if err != nil {
		g.sendUserMessage(actorCtx, fmt.Sprintf("error importing: %v", err))
	}
}

func (g *Game) handleUserInput(actorCtx actor.Context, input *jasonsgame.UserInput) {
	if sender := actorCtx.Sender(); sender != nil {
		log.Debugf("responding to parent with CommandReceived")
		actorCtx.Respond(&jasonsgame.CommandReceived{Sequence: g.messageSequence})
		g.messageSequence++
	}

	cmd, args := g.commands.findCommand(input.Message)
	if cmd == nil {
		g.sendUserMessage(actorCtx, "I'm sorry I don't understand.")
		return
	}

	var err error
	log.Debugf("received command %v", cmd.Name())
	switch cmd.Name() {
	case "exit":
		g.sendUserMessage(actorCtx, "exit is unsupported in the browser")
	case "set-description":
		err = g.handleSetDescription(actorCtx, args)
	case "tip-zoom":
		err = g.handleTipZoom(actorCtx, args)
	case "location-zoom":
		err = g.handleLocationZoom(actorCtx, args)
	case "refresh":
		err = g.refreshAllInteractions(actorCtx)
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
	case "player-inventory-list":
		err = g.handlePlayerInventoryList(actorCtx)
	case "location-inventory-list":
		err = g.handleLocationInventoryList(actorCtx)
	case "create-location":
		err = g.handleCreateLocation(actorCtx, args)
	case "connect-location":
		err = g.handleConnectLocation(actorCtx, args)
	case "help":
		g.sendUserMessage(actorCtx, "available commands:")
		for _, c := range g.commands {
			g.sendUserMessage(actorCtx, c.Parse())
		}
	case "name":
		err = g.handleName(args)
	case "interaction":
		err = g.handleInteractionInput(actorCtx, cmd.(*interactionCommand), args)
	default:
		log.Error("unhandled but matched command", cmd.Name())
	}
	if err != nil {
		g.sendUserMessage(actorCtx, fmt.Sprintf("error with your command: %v", err))
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
		return errors.Wrap(err, "error building portal")
	}

	if respErr := response.(*BuildPortalResponse).Error; respErr != nil {
		return errors.Wrap(err, "error building portal")
	}

	g.sendUserMessage(actorCtx, fmt.Sprintf("successfully built a portal to %s", toDid))
	return nil
}

func (g *Game) handleTipZoom(actorCtx actor.Context, tip string) error {
	tipCid, err := cid.Parse(tip)
	if err != nil {
		g.sendUserMessage(actorCtx, fmt.Sprintf("error parsing tip (%s): %v", tip, err))
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

func (g *Game) handleLocationZoom(actorCtx actor.Context, did string) error {
	tree, err := g.network.GetTree(did)
	if err != nil {
		return fmt.Errorf("error fetching target location: %v", err)
	}
	if tree == nil {
		return fmt.Errorf("could not find target location")
	}

	g.setLocation(actorCtx, tree.MustId())
	g.sendUILocation(actorCtx)
	return nil
}

func (g *Game) handleSetDescription(actorCtx actor.Context, desc string) error {
	response, err := actorCtx.RequestFuture(g.locationActor, &SetLocationDescriptionRequest{Description: desc}, 5*time.Second).Result()

	if err != nil {
		return errors.Wrap(err, "error setting description")
	}

	descriptionResponse, ok := response.(*SetLocationDescriptionResponse)

	if !ok || descriptionResponse.Error != nil {
		return errors.Wrap(descriptionResponse.Error, "error setting description")
	}

	g.sendUILocation(actorCtx)
	return nil
}

func (g *Game) handleInteractionInput(actorCtx actor.Context, cmd *interactionCommand, args string) error {
	var err error

	switch interaction := cmd.interaction.(type) {
	case *RespondInteraction:
		g.sendUserMessage(actorCtx, interaction.Response)
	case *ChangeLocationInteraction:
		g.setLocation(actorCtx, interaction.Did)
		g.sendUILocation(actorCtx)
	case *DropObjectInteraction:
		err = g.handleDropObject(actorCtx, interaction)
	case *PickUpObjectInteraction:
		err = g.handlePickUpObject(actorCtx, interaction)
	case *GetTreeValueInteraction:
		err = g.handleGetTreeValueInteraction(actorCtx, interaction)
	case *CipherInteraction:
		nextInteraction, _, err := interaction.Unseal(args)
		if err != nil {
			return err
		}
		nextCmd := &interactionCommand{
			parse:       nextInteraction.GetCommand(),
			interaction: nextInteraction,
		}
		return g.handleInteractionInput(actorCtx, nextCmd, "")
	default:
		g.sendUserMessage(actorCtx, fmt.Sprintf("no interaction matching %s, type %v", cmd.Parse(), reflect.TypeOf(interaction)))
	}

	return err
}

func (g *Game) handleGetTreeValueInteraction(actorCtx actor.Context, interaction *GetTreeValueInteraction) error {
	ctx := context.TODO()

	tree, err := g.network.GetTree(interaction.Did)
	if err != nil {
		return errors.Wrap(err, "error fetching tree")
	}
	if tree == nil {
		return fmt.Errorf("could not find tree with did %v", interaction.Did)
	}

	pathSlice, err := consensus.DecodePath(interaction.Path)
	if err != nil {
		return errors.Wrap(err, "error casting path")
	}

	value, _, err := tree.ChainTree.Dag.Resolve(ctx, append([]string{"tree", "data", "jasons-game"}, pathSlice...))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error fetching value for %v", pathSlice))
	}

	g.sendUserMessage(actorCtx, value)
	return nil
}

func (g *Game) handleDropObject(actorCtx actor.Context, interaction *DropObjectInteraction) error {
	response, err := actorCtx.RequestFuture(g.inventoryActor, &TransferObjectRequest{
		Did: interaction.Did,
		To:  g.locationDid,
	}, 5*time.Second).Result()

	if err != nil {
		return errors.Wrap(err, "error executing drop request")
	}

	resp, ok := response.(*TransferObjectResponse)
	if !ok {
		return fmt.Errorf("error casting drop object response")
	}

	if resp.Error != nil {
		return resp.Error
	}

	g.sendUserMessage(actorCtx, "object has been dropped into your current location")
	return nil
}

func (g *Game) handlePickUpObject(actorCtx actor.Context, interaction *PickUpObjectInteraction) error {
	response, err := actorCtx.RequestFuture(g.locationActor, &TransferObjectRequest{
		Did: interaction.Did,
		To:  g.playerTree.Did(),
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

	g.sendUserMessage(actorCtx, "object has been picked up")
	return nil
}

func (g *Game) createObject(actorCtx actor.Context, objName string, desc string) (*CreateObjectResponse, error) {
	response, err := actorCtx.RequestFuture(g.inventoryActor, &CreateObjectRequest{
		Name:        objName,
		Description: desc,
	}, 5*time.Second).Result()
	if err != nil {
		return nil, err
	}

	newObject, ok := response.(*CreateObjectResponse)
	if !ok {
		return nil, fmt.Errorf("error casting create object response")
	}

	return newObject, nil
}

func (g *Game) handleCreateObject(actorCtx actor.Context, args string) error {
	splitArgs := strings.Split(args, " ")
	objName := splitArgs[0]
	desc := strings.Join(splitArgs[1:], " ")

	newObject, err := g.createObject(actorCtx, objName, desc)
	if err != nil {
		return err
	}

	g.sendUserMessage(actorCtx, fmt.Sprintf("%s has been created with DID %s and is in your bag of hodling", objName, newObject.Object.Did))
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
		g.sendUserMessage(actorCtx, "your bag of hodling appears to be empty")
		return nil
	}

	g.sendUserMessage(actorCtx, "inside of your bag of hodling you find:")
	for objName, obj := range inventoryList.Objects {
		g.sendUserMessage(actorCtx, fmt.Sprintf("%s (%s)", objName, obj.Did))
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
		g.sendUserMessage(actorCtx, "you see the following objects around you:")
		for objName, obj := range inventoryList.Objects {
			g.sendUserMessage(actorCtx, fmt.Sprintf("%s (%s)", objName, obj.Did))
		}
	}

	if l.Portal != nil {
		sawSomething = true
		g.sendUserMessage(actorCtx, fmt.Sprintf("you see a mysterious portal leading to %s", l.Portal.To))
	}

	if !sawSomething {
		g.sendUserMessage(actorCtx, "you look around but don't see anything")
	}
	return nil
}

func (g *Game) createLocation(actorCtx actor.Context) (*consensus.SignedChainTree, error) {
	return g.network.CreateChainTree()
}

func (g *Game) handleCreateLocation(actorCtx actor.Context, args string) error {
	newLocation, err := g.createLocation(actorCtx)
	if err != nil {
		return err
	}

	g.sendUserMessage(actorCtx, "new location created "+newLocation.MustId())
	return nil
}

func (g *Game) connectLocations(actorCtx actor.Context, toDid string, interactionCommand string) error {
	targetTree, err := g.network.GetTree(toDid)
	if err != nil {
		return fmt.Errorf("error fetching target location: %v", err)
	}
	if targetTree == nil {
		return fmt.Errorf("could not find target location")
	}

	loc := NewLocationTree(g.network, targetTree)

	auths, err := g.playerTree.Authentications()
	if err != nil {
		return fmt.Errorf("error fetching player authentications")
	}
	isOwnedBy, _ := loc.IsOwnedBy(auths)
	if !isOwnedBy {
		return fmt.Errorf("can't connect a location that you don't own")
	}

	interaction := &ChangeLocationInteraction{
		Command: interactionCommand,
		Did:     toDid,
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

	g.sendUserMessage(actorCtx, fmt.Sprintf("added a connection to %s as %s", toDid, interactionCommand))
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

	return g.connectLocations(actorCtx, toDid, interactionCommand)
}

func (g *Game) sendUILocation(actorCtx actor.Context) {
	l, err := g.getCurrentLocation(actorCtx)

	if err != nil {
		g.sendUserMessage(actorCtx, fmt.Errorf("error getting current location: %v", err))
	}

	g.sendUserMessage(actorCtx, l)
}

func (g *Game) sendUserMessage(actorCtx actor.Context, mesgInter interface{}) {
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

func (g *Game) sendImportResult(actorCtx actor.Context, msg proto.Message) {
	var result *jasonsgame.ImportResult

	switch msg := msg.(type) {
	case *jasonsgame.LocationSpec:
		result = &jasonsgame.ImportResult{
			Spec: &jasonsgame.ImportResult_Location{Location: msg},
		}

	case *jasonsgame.ObjectSpec:
		result = &jasonsgame.ImportResult{
			Spec: &jasonsgame.ImportResult_Object{Object: msg},
		}

	case *jasonsgame.PopulateSpec:
		result = &jasonsgame.ImportResult{
			Spec: &jasonsgame.ImportResult_Populate{Populate: msg},
		}

	default:
		log.Errorf("error, unrecognized import result: %v", msg)
	}

	actorCtx.Send(g.ui, result)
}

func (g *Game) sendCommandUpdate(actorCtx actor.Context) {
	parsedCommands := make([]string, len(g.commands))
	for i, c := range g.commands {
		parsedCommands[i] = c.Parse()
	}

	cmdUpdate := &jasonsgame.CommandUpdate{Commands: parsedCommands}

	actorCtx.Send(g.ui, cmdUpdate)
}

func (g *Game) setLocation(actorCtx actor.Context, locationDid string) {
	oldLocationActor := g.locationActor
	if oldLocationActor != nil {
		actorCtx.Stop(g.locationActor)
	}
	g.locationActor = actorCtx.Spawn(NewLocationActorProps(&LocationActorConfig{
		Network: g.network,
		Did:     locationDid,
	}))
	g.locationDid = locationDid

	err := g.replaceInteractionsFor(actorCtx, g.locationActor, oldLocationActor)
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

func (g *Game) setCommands(actorCtx actor.Context, newCommands commandList) {
	g.commands = newCommands
	g.sendCommandUpdate(actorCtx)
}

func (g *Game) refreshAllInteractions(actorCtx actor.Context) error {
	for pid := range g.commandsByActorCache {
		err := g.refreshInteractionsFor(actorCtx, pid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) replaceInteractionsFor(actorCtx actor.Context, pid *actor.PID, oldPid *actor.PID) error {
	if oldPid != nil {
		delete(g.commandsByActorCache, oldPid)
	}
	return g.refreshInteractionsFor(actorCtx, pid)
}

func (g *Game) refreshInteractionsFor(actorCtx actor.Context, pid *actor.PID) error {
	if g.commandsByActorCache == nil {
		g.commandsByActorCache = make(map[*actor.PID]commandList)
	}
	var err error
	g.commandsByActorCache[pid], err = g.interactionCommandsFor(actorCtx, pid)

	if err != nil {
		return err
	}

	newCommands := defaultCommandList
	for _, commands := range g.commandsByActorCache {
		newCommands = append(newCommands, commands...)
	}

	g.setCommands(actorCtx, newCommands)
	return nil
}

func (g *Game) interactionCommandsFor(actorCtx actor.Context, pid *actor.PID) (commandList, error) {
	response, err := actorCtx.RequestFuture(pid, &ListInteractionsRequest{}, 5*time.Second).Result()
	if err != nil || response == nil {
		return nil, fmt.Errorf("error fetching interactions %v", err)
	}

	interactionsResponse, ok := response.(*ListInteractionsResponse)
	if !ok {
		return nil, fmt.Errorf("error casting ListInteractionsResponse")
	}
	if interactionsResponse.Error != nil {
		return nil, interactionsResponse.Error
	}

	interactions := interactionsResponse.Interactions
	interactionCommands := make(commandList, len(interactions))
	for i, interaction := range interactions {
		interactionCommands[i] = &interactionCommand{
			parse:       interaction.GetCommand(),
			interaction: interaction,
		}
	}
	return interactionCommands, nil
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

func (g *Game) handleStateChange(actorCtx actor.Context, msg *StateChange) {
	pid := msg.PID
	if _, ok := g.commandsByActorCache[pid]; ok {
		err := g.refreshInteractionsFor(actorCtx, pid)
		if err != nil {
			log.Warningf("error refreshing interactions on state change %v", err)
		}
	}
}
