package game

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/cache"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/game/static"
	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
	"github.com/quorumcontrol/jasons-game/inkfaucet/invites"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/jasons-game/utils/stringslice"
	"github.com/quorumcontrol/messages/build/go/signatures"
)

var log = logging.Logger("game")

const hasArtifactHint = "-"

type ping struct{}

type indentedList []string

type Game struct {
	ui                   *actor.PID
	network              network.Network
	playerTree           *PlayerTree
	commands             commandList
	messageSequence      uint64
	locationDid          string
	locationActor        *actor.PID
	inventoryActor       *actor.PID
	inventoryHandler     *PlayerInventoryHandler
	commandsByActorCache map[*actor.PID]commandList
	behavior             actor.Behavior
	inkDID               string
	invitesActor         *actor.PID
	ds                   datastore.Batching
}

type GameConfig struct {
	PlayerTree *PlayerTree
	UiActor    *actor.PID
	Network    network.Network
	InkDID     string
	DataStore  datastore.Batching
}

type StateChange struct {
	PID *actor.PID
}

var lastLocationKey = datastore.NewKey("last-location")

func NewGameProps(cfg *GameConfig) *actor.Props {
	g := &Game{
		ui:         cfg.UiActor,
		network:    cfg.Network,
		commands:   defaultCommandList,
		playerTree: cfg.PlayerTree,
		behavior:   actor.NewBehavior(),
		inkDID:     cfg.InkDID,
		ds:         cfg.DataStore,
	}

	if g.ds == nil {
		g.ds = config.MemoryDataStore()
	}

	if g.playerTree == nil {
		g.behavior.Become(g.ReceiveInvitation)
	} else {
		g.behavior.Become(g.ReceiveGame)
	}

	return actor.PropsFromProducer(func() actor.Actor {
		return g
	})
}

func (g *Game) Receive(actorCtx actor.Context) {
	log.Debugf("received message to dispatch to current behavior: %+v", actorCtx.Message())
	g.behavior.Receive(actorCtx)
}

func (g *Game) ReceiveInvitation(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		log.Debug("actor started in invitation mode")
		g.initializeInvitation(actorCtx)
	case *actor.Stopping:
		log.Debug("stopping actor in invitation mode; poisoning invite actor too")
		actorCtx.Poison(g.invitesActor)
	case *jasonsgame.UserInput:
		log.Debugf("actor received user input in invitation mode: %+v", msg)
		g.handleInvitationInput(actorCtx, msg)
	case *jasonsgame.CommandUpdate:
		log.Debugf("received command update request in invitation mode: %+v", msg)
		g.sendInvitationCommandUpdate(actorCtx)
	case *ping:
		actorCtx.Respond(true)
	case *actor.Terminated:
		log.Info("actor terminated in invitation mode")
	default:
		log.Warningf("received message of unrecognized type %T in invitation mode: %+v", msg, msg)
	}
}

func (g *Game) ReceiveGame(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		log.Debug("actor started in game mode")
		g.initializeGame(actorCtx)
	case *jasonsgame.UserInput:
		log.Debugf("actor received user input: %+v", msg)
		g.handleUserInput(actorCtx, msg)
	case *jasonsgame.CommandUpdate:
		log.Debugf("actor received command update request: %+v", msg)
		g.sendCommandUpdate(actorCtx)
	case *StateChange:
		log.Debugf("actor received state change message: %+v", msg)
		g.handleStateChange(actorCtx, msg)
	case *ping:
		actorCtx.Respond(true)
	case *actor.Terminated:
		log.Infof("actor terminated: %s", msg)
	default:
		log.Warningf("received message of unrecognized type %T: %+v", msg, msg)
	}
}

func (g *Game) initializeCommon(actorCtx actor.Context) {
	actorCtx.Send(g.ui, &ui.SetGame{Game: actorCtx.Self()})
}

func (g *Game) initializeInvitation(actorCtx actor.Context) {
	log.Debug("initializing game actor in invitation mode")

	g.initializeCommon(actorCtx)

	invitesActor := invites.NewInvitesActor(context.TODO(), invites.InvitesActorConfig{
		Net:    g.network,
		InkDID: g.inkDID,
	})
	invitesActor.Start(actor.EmptyRootContext)
	g.invitesActor = invitesActor.PID()

	g.sendInvitationCommandUpdate(actorCtx)

	g.sendUserMessage(actorCtx, "Welcome to Jason's Game! Please enter your invite code like this: `invitation [code]`.")
}

func (g *Game) loadCache() {
	key := datastore.NewKey("cacheLoaded")
	cacheLoaded, err := g.ds.Get(key)
	if err != nil && err != datastore.ErrNotFound {
		log.Warning(errors.Wrap(err, "error loading cache check key"))
		return
	}
	if string(cacheLoaded) != "true" {
		err := cache.Load(g.network.Ipld())
		if err != nil {
			log.Warning(err)
			return
		}
		err = g.ds.Put(key, []byte("true"))
		if err != nil {
			log.Warning(err)
		}
	}
}

func (g *Game) initializeGame(actorCtx actor.Context) {
	log.Debug("initializing game actor in game mode")

	// Only load cache when its a remote network
	switch g.network.(type) {
	case *network.RemoteNetwork:
		go g.loadCache()
	}

	g.initializeCommon(actorCtx)

	g.inventoryHandler = NewPlayerInventoryHandler(g.network, g.playerTree.Did())

	g.inventoryActor = actorCtx.Spawn(NewInventoryActorProps(&InventoryActorConfig{
		Did:     g.playerTree.Did(),
		Network: g.network,
		Handler: g.inventoryHandler,
	}))
	err := g.refreshInteractionsFor(actorCtx, g.inventoryActor)
	if err != nil {
		panic(errors.Wrap(err, "error attaching interactions for inventory"))
	}

	g.setLocation(actorCtx, g.getDefaultLocation())

	g.sendUserMessage(actorCtx, fmt.Sprintf("Welcome Player %s", g.playerTree.Did()))

	if flash, _ := static.Get(g.network, "FlashMessage"); len(flash) > 0 {
		g.sendUserMessage(actorCtx, flash)
	}

	l, err := g.getCurrentLocation(actorCtx)

	if err != nil {
		panic(err)
	}

	g.sendUserMessage(actorCtx, l)
}

// Try to get the last visited location, else go
// to ArcadiaStart, else go to player home
func (g *Game) getDefaultLocation() string {
	locDidBytes, _ := g.ds.Get(lastLocationKey)
	if locDidBytes != nil {
		return string(locDidBytes)
	}

	locDid, _ := static.Get(g.network, "ArcadiaStart")
	if locDid != "" {
		return locDid
	}

	return g.playerTree.HomeLocation.MustId()
}

func (g *Game) acknowledgeReceipt(actorCtx actor.Context) {
	if sender := actorCtx.Sender(); sender != nil {
		log.Debugf("responding to parent with CommandReceived")
		actorCtx.Respond(&jasonsgame.CommandReceived{Sequence: g.messageSequence})
		g.messageSequence++
	}
}

func (g *Game) handleInvitationInput(actorCtx actor.Context, input *jasonsgame.UserInput) {
	g.acknowledgeReceipt(actorCtx)

	cmdComponents := strings.Split(input.Message, " ")
	switch cmd := cmdComponents[0]; cmd {
	case "invitation":
		log.Debug("received invite submission")

		inviteSubmission := &inkfaucet.InviteSubmission{
			Invite: cmdComponents[1],
		}

		log.Debug("sending invite code to invites actor")

		req := actorCtx.RequestFuture(g.invitesActor, inviteSubmission, 10*time.Second)

		uncastInviteResp, err := req.Result()
		if err != nil {
			panic("invalid invite code")
		}

		log.Debugf("received response from invites actor: %+v", uncastInviteResp)

		inviteResp, ok := uncastInviteResp.(*inkfaucet.InviteSubmissionResponse)
		if !ok {
			panic("invalid invite code")
		}

		if inviteResp.GetError() != "" {
			panic("invalid invite code")
		}

		g.sendUserMessage(actorCtx, "Invite code accepted. Starting game...")

		log.Debug("creating player tree")

		playerTree, err := CreatePlayerTree(g.network, inviteResp.PlayerChainId)
		if err != nil {
			panic("error creating player tree")
		}

		g.playerTree = playerTree

		log.Debug("putting game actor into game mode")

		g.behavior.Become(g.ReceiveGame)

		log.Debug("initializing game mode")

		g.initializeGame(actorCtx)

	case "help":
		g.sendUserMessage(actorCtx, "available commands:")
		g.sendUserMessage(actorCtx, "help")
		g.sendUserMessage(actorCtx, "invitation")
	}
}

func (g *Game) handleUserInput(actorCtx actor.Context, input *jasonsgame.UserInput) {
	g.acknowledgeReceipt(actorCtx)

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
	case "tip-zoom":
		err = g.handleTipZoom(actorCtx, args)
	case "refresh":
		err = g.refreshAllInteractions(actorCtx)
		g.sendUILocation(actorCtx)
	case "create-object":
		err = g.handleCreateObjectFromArgs(actorCtx, args)
	case "player-inventory-list":
		err = g.handlePlayerInventoryList(actorCtx)
	case "location-inventory-list":
		err = g.handleLocationInventoryList(actorCtx)
	case "create-location":
		err = g.handleCreateLocation(actorCtx, args)
	case "connect-location":
		err = g.handleConnectLocation(actorCtx, args)
	case "transfer-object":
		err = g.handleTransferObjectCmd(actorCtx, args)
	case "receive-object":
		err = g.handleReceiveObjectCmd(actorCtx, args)
	case "help":
		err = g.handleHelp(actorCtx, args)
	case "interaction":
		err = g.handleInteractionInput(actorCtx, cmd.(*interactionCommand), args)
	default:
		log.Error("unhandled but matched command", cmd.Name())
	}
	if err != nil {
		g.sendUserMessage(actorCtx, fmt.Sprintf("error with your command: %v", err))
	}
}

var transferRegex = regexp.MustCompile(`(.*) to (did:tupelo:[0-9A-Za-z]{42}){1}\s?$`)

func (g *Game) handleTransferObjectCmd(actorCtx actor.Context, args string) error {
	matches := transferRegex.FindStringSubmatch(args)
	if len(matches) < 3 {
		return fmt.Errorf("transfer object requires the following syntax:\n\n`transfer object {object name} to {player DID}`")
	}

	objectName := strings.TrimSpace(matches[1])
	targetDid := matches[2]

	inventoryList, err := g.getInventoryList(actorCtx, g.inventoryActor)
	if err != nil {
		return fmt.Errorf("error getting player inventory list: %v", err)
	}

	if len(inventoryList.Objects) == 0 {
		g.sendUserMessage(actorCtx, fmt.Sprintf("can't transfer %s, your bag of hodling appears to be empty", objectName))
		return nil
	}

	var objectDid string

	for invObjName, invObj := range inventoryList.Objects {
		if invObjName == objectName {
			objectDid = invObj.Did
			break
		}
	}

	if len(objectDid) == 0 {
		g.sendUserMessage(actorCtx, fmt.Sprintf("can't transfer %s, its not in your bag of hodling", objectName))
		return nil
	}

	// This is a workaround to avoid rewriting the inventory handlers to deal with
	// "unacked" transfers. Essentially this just waits till the inventory handler
	// does its chown to the new owner, then sets an additional attribute here,
	// which then triggers the UnrestrictedRemoveHandlers "ack", allowing the
	// TransferObjectRequest future to complete below
	go func() {
		future := actor.NewFuture(30 * time.Second)
		pid := actorCtx.Spawn(actor.PropsFromFunc(func(subActorCtx actor.Context) {
			switch msg := subActorCtx.Message().(type) {
			case *actor.Started:
				subActorCtx.Spawn(g.network.NewCurrentStateSubscriptionProps(objectDid))
			case *signatures.CurrentState:
				subActorCtx.Send(future.PID(), msg)
			}
		}))
		defer actorCtx.Stop(pid)

		// object changed owners inside UnrestrictedRemoveHandler
		_, err := future.Result()
		if err != nil {
			log.Error(err)
			return
		}

		newTree, err := g.network.GetTree(objectDid)
		if err != nil {
			log.Error(err)
			return
		}

		_, err = g.network.UpdateChainTree(newTree, "jasons-game/transferred-from", g.playerTree.tree.MustId())
		if err != nil {
			log.Error(err)
		}
	}()

	response, err := actorCtx.RequestFuture(g.inventoryActor, &TransferObjectRequest{
		Did: objectDid,
		To:  targetDid,
	}, 30*time.Second).Result()

	if err != nil {
		return errors.Wrap(err, "error executing transfer object request")
	}

	resp, ok := response.(*TransferObjectResponse)
	if !ok {
		return fmt.Errorf("error casting transfer object response")
	}

	if resp.Error != nil {
		return resp.Error
	}

	g.sendUserMessage(actorCtx, fmt.Sprintf("%s has been sent for transfer - the receiving player must now type:\n\n`receive object %s`\n\nif you wish to cancel the transfer, you may run the preceding command to add the object back to your bag of hodling", objectName, objectDid))
	return nil
}

func (g *Game) handleReceiveObjectCmd(actorCtx actor.Context, args string) error {
	objectDid := args

	if matched, _ := regexp.MatchString(`did:tupelo:[0-9A-Za-z]{42}`, objectDid); !matched {
		return fmt.Errorf("must specify an DID to recieve object")
	}

	object, err := FindObjectTree(g.network, objectDid)
	if err != nil {
		return errors.Wrap(err, "object not found")
	}

	if object == nil {
		g.sendUserMessage(actorCtx, fmt.Sprintf("object %s not found", objectDid))
		return nil
	}

	objName, err := object.GetName()
	if err != nil {
		return errors.Wrap(err, "error getting object name")
	}

	changeEventCh := make(chan *InventoryChangeEvent, 1)
	subscription := g.inventoryHandler.Subscribe(objectDid, func(evt *InventoryChangeEvent) {
		changeEventCh <- evt
	})
	defer g.inventoryHandler.Unsubscribe(subscription)

	g.inventoryHandler.ExpectObject(objectDid)

	actorCtx.Send(g.inventoryActor, &jasonsgame.TransferredObjectMessage{
		To:     g.playerTree.Did(),
		Object: objectDid,
	})

	changeEvent := <-changeEventCh

	if changeEvent.Error != "" {
		return fmt.Errorf(changeEvent.Error)
	}

	g.sendUserMessage(actorCtx, fmt.Sprintf("%s (%s) has been transferred into your bag of hodling", objName, objectDid))
	return nil
}

func (g *Game) handleHelp(actorCtx actor.Context, args string) error {
	toSend := []string{}

	for _, c := range g.commands {
		if !c.Hidden() && c.HelpGroup() == args && !stringslice.Include(toSend, c.Parse()) {
			toSend = append(toSend, c.Parse())
		}
	}

	if len(toSend) == 0 {
		g.sendUserMessage(actorCtx, fmt.Sprintf("Sorry, I am not sure how I can help with '%s'...\n"+
			"Maybe you can try looking around, asking for help on the location or help on an object.", args))
		return nil
	}

	sort.Slice(toSend, func(i, j int) bool {
		// push help commands down
		if strings.HasPrefix(toSend[i], "help") {
			return false
		}
		if strings.HasPrefix(toSend[j], "help") {
			return true
		}
		return toSend[i] < toSend[j]
	})

	g.sendUserMessage(actorCtx, append(indentedList{"available commands:"}, toSend...))
	return nil
}

func (g *Game) handleBuildPortal(actorCtx actor.Context, toDid string) error {
	response, err := actorCtx.RequestFuture(g.locationActor, &BuildPortalRequest{
		To: toDid,
	}, 30*time.Second).Result()
	if err != nil {
		return errors.Wrap(err, "error building portal")
	}

	if respErr := response.(*BuildPortalResponse).Error; respErr != nil {
		return errors.Wrap(err, "error building portal")
	}

	g.sendUserMessage(actorCtx, fmt.Sprintf("successfully built a portal to %s", toDid))
	return nil
}

func (g *Game) handleDeletePortal(actorCtx actor.Context, toDid string) error {
	response, err := actorCtx.RequestFuture(g.locationActor, &DeletePortalRequest{}, 30*time.Second).Result()
	if err != nil {
		return errors.Wrap(err, "error deleting portal")
	}

	if respErr := response.(*DeletePortalResponse).Error; respErr != nil {
		return errors.Wrap(err, "error deleting portal")
	}

	g.sendUserMessage(actorCtx, "successfully deleted the portal")
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

func (g *Game) handleInteractionInput(actorCtx actor.Context, cmd *interactionCommand, args string) error {
	var err error

	log.Debugf("handling interaction type %T", cmd.interaction)

	switch interaction := cmd.interaction.(type) {
	case *RespondInteraction:
		g.sendUserMessage(actorCtx, interaction.Response)
	case *BuildPortalInteraction:
		err = g.handleBuildPortal(actorCtx, args)
	case *DeletePortalInteraction:
		err = g.handleDeletePortal(actorCtx, args)
	case *ChangeLocationInteraction:
		g.handleChangeLocation(actorCtx, interaction.Did)
	case *ChangeNamedLocationInteraction:
		g.handleChangeNamedLocation(actorCtx, interaction.Name)
	case *DropObjectInteraction:
		err = g.handleDropObject(actorCtx, cmd, interaction)
	case *PickUpObjectInteraction:
		err = g.handlePickUpObject(actorCtx, interaction)
	case *GetTreeValueInteraction:
		err = g.handleGetTreeValueInteraction(actorCtx, interaction)
	case *SetTreeValueInteraction:
		err = g.handleSetTreeValueInteraction(actorCtx, interaction, args)
	case *LookAroundInteraction:
		err = g.handleLocationInventoryList(actorCtx)
	case *CreateObjectInteraction:
		err = g.handleCreateObjectInteraction(actorCtx, interaction)
	case *CipherInteraction:
		nextInteraction, _, err := interaction.Unseal(args)
		if err != nil {
			return err
		}
		nextCmd := &interactionCommand{
			parse:       nextInteraction.GetCommand(),
			interaction: nextInteraction,
		}
		return g.handleInteractionInput(actorCtx, nextCmd, args)
	case *ChainedInteraction:
		interactions, err := interaction.Interactions()
		if err != nil {
			return err
		}
		for _, nextInteraction := range interactions {
			nextCmd := &interactionCommand{
				parse:       interaction.GetCommand(),
				interaction: nextInteraction,
			}
			err = g.handleInteractionInput(actorCtx, nextCmd, args)
			if err != nil {
				return err
			}
		}
	default:
		g.sendUserMessage(actorCtx, fmt.Sprintf("no interaction matching %s, type %v", cmd.Parse(), reflect.TypeOf(interaction)))
	}

	return err
}

func (g *Game) handleChangeLocation(actorCtx actor.Context, did string) {
	log.Debugf("setting new location to %s", did)
	g.setLocation(actorCtx, did)

	// we can ignore the error here, if it fails, no big deal since its just
	// a hint that an artifact exists
	inventoryList, _ := g.getInventoryList(actorCtx, g.locationActor)

	log.Debug("sending new location to UI")
	g.sendUILocation(actorCtx)
	g.sendArtifactHint(actorCtx, inventoryList)
}

func (g *Game) handleChangeNamedLocation(actorCtx actor.Context, name string) {
	var did string

	switch name {
	case "last-location":
		locDidBytes, _ := g.ds.Get(lastLocationKey)
		if locDidBytes != nil {
			did = string(locDidBytes)
		}
		if did == "" {
			did, _ = static.Get(g.network, "ArcadiaStartAgain")
		}
	case "home":
		did = g.playerTree.HomeLocation.MustId()
	default:
		did, _ = static.Get(g.network, name)
	}

	if did == "" {
		did = g.playerTree.HomeLocation.MustId()
	}

	g.handleChangeLocation(actorCtx, did)
}

func (g *Game) handleSetTreeValueInteraction(actorCtx actor.Context, interaction *SetTreeValueInteraction, args string) error {
	ctx := context.TODO()

	tree, err := g.network.GetTree(interaction.Did)
	if err != nil {
		return errors.Wrap(err, "error fetching tree")
	}
	if tree == nil {
		return fmt.Errorf("could not find tree with did %v", interaction.Did)
	}

	_, err = interaction.SetValue(ctx, g.network, tree, args)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error setting value on tree %v", interaction.Did))
	}

	g.sendUserMessage(actorCtx, fmt.Sprintf("set %v to %v", interaction.Path, args))

	return nil
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
		return errors.Wrap(err, fmt.Sprintf("error fetching value for %v", interaction.Path))
	}

	var toSend string
	switch msg := value.(type) {
	case string:
		toSend = msg
	case []interface{}:
		stringSlice := make([]string, len(msg))
		for i, v := range msg {
			stringSlice[i] = fmt.Sprintf("%v", v)
		}
		toSend = strings.Join(stringSlice, "\n")
	default:
		valBytes, err := yaml.Marshal(msg)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error casting value at %v", interaction.Path))
		}
		toSend = string(valBytes)
	}
	g.sendUserMessage(actorCtx, toSend)
	return nil
}

func (g *Game) handleDropObject(actorCtx actor.Context, cmd *interactionCommand, interaction *DropObjectInteraction) error {
	if interaction.Did != cmd.did {
		return fmt.Errorf("Interaction from %s tried to drop %s - this is not allowed", cmd.did, interaction.Did)
	}

	locationInventoryDid, err := actorCtx.RequestFuture(g.locationActor, &GetInventoryDid{}, 5*time.Second).Result()
	if err != nil {
		return errors.Wrap(err, "error executing drop request")
	}

	response, err := actorCtx.RequestFuture(g.inventoryActor, &TransferObjectRequest{
		Did: interaction.Did,
		To:  locationInventoryDid.(string),
	}, 30*time.Second).Result()

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
	changeEventCh := make(chan *InventoryChangeEvent, 1)

	objectDid := interaction.Did
	subscription := g.inventoryHandler.Subscribe(objectDid, func(evt *InventoryChangeEvent) {
		changeEventCh <- evt
	})
	defer g.inventoryHandler.Unsubscribe(subscription)

	g.inventoryHandler.ExpectObject(objectDid)

	response, err := actorCtx.RequestFuture(g.locationActor, &TransferObjectRequest{
		Did: interaction.Did,
		To:  g.playerTree.Did(),
	}, 30*time.Second).Result()

	if err != nil {
		return err
	}

	resp, ok := response.(*TransferObjectResponse)
	if !ok {
		return fmt.Errorf("error casting pick up object response")
	}

	if resp.Error == ErrExists {
		objectTree, err := FindObjectTree(g.network, objectDid)
		if err != nil {
			return err
		}
		objName, _ := objectTree.GetName() // nolint: golint
		if objName == "" {
			objName = "object"
		}
		g.sendUserMessage(actorCtx, fmt.Sprintf("'%s' is already in your bag of hodling", objName))
		return nil
	}

	if resp.Error != nil {
		g.sendUserMessage(actorCtx, resp.Error.Error())
		return nil
	}

	changeEvent := <-changeEventCh

	if changeEvent.Error != "" {
		return fmt.Errorf(changeEvent.Error)
	}

	if changeEvent.Message != "" {
		g.sendUserMessage(actorCtx, changeEvent.Message)
	} else {
		g.sendUserMessage(actorCtx, "object has been picked up")
	}

	return nil
}

func (g *Game) objectAlreadyExistsResponse(actorCtx actor.Context, objName string) {
	g.sendUserMessage(actorCtx,
		fmt.Sprintf("You already have an object named \"%s\" in your bag of hodling.",
			objName))
}

func (g *Game) handleCreateObjectInteraction(actorCtx actor.Context, interaction *CreateObjectInteraction) error {
	err := g.handleCreateObjectRequest(actorCtx, &CreateObjectRequest{
		Name:             interaction.Name,
		Description:      interaction.Description,
		WithInscriptions: interaction.WithInscriptions,
	})

	if err == ErrExists {
		g.objectAlreadyExistsResponse(actorCtx, interaction.Name)
		return nil
	}

	return err
}

func (g *Game) handleCreateObjectFromArgs(actorCtx actor.Context, args string) error {
	splitArgs := strings.Split(args, " ")
	name := splitArgs[0]
	err := g.handleCreateObjectRequest(actorCtx, &CreateObjectRequest{
		Name:        name,
		Description: strings.Join(splitArgs[1:], " "),
	})

	if err == ErrExists {
		g.objectAlreadyExistsResponse(actorCtx, name)
		return nil
	}

	return err
}

func (g *Game) handleCreateObjectRequest(actorCtx actor.Context, req *CreateObjectRequest) error {
	response, err := actorCtx.RequestFuture(g.inventoryActor, req, 30*time.Second).Result()
	if err != nil {
		return err
	}

	createObjectResp, ok := response.(*CreateObjectResponse)
	if !ok {
		return fmt.Errorf("error casting create object response")
	}

	if createObjectResp.Error != nil {
		return createObjectResp.Error
	}

	newObject := createObjectResp.Object

	g.sendUserMessage(actorCtx, fmt.Sprintf("%s has been created with DID %s and is in your bag of hodling", req.Name, newObject.Did))
	return nil
}

func (g *Game) getInventoryList(actorCtx actor.Context, pid *actor.PID) (*InventoryListResponse, error) {
	response, err := actorCtx.RequestFuture(pid, &InventoryListRequest{}, 30*time.Second).Result()
	if err != nil {
		return nil, err
	}

	inventoryList, ok := response.(*InventoryListResponse)
	if !ok {
		return nil, fmt.Errorf("error casting InventoryListResponse")
	}

	return inventoryList, nil
}

func (g *Game) handlePlayerInventoryList(actorCtx actor.Context) error {
	inventoryList, err := g.getInventoryList(actorCtx, g.inventoryActor)
	if err != nil {
		return fmt.Errorf("error getting player inventory list: %v", err)
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
	inventoryList, err := g.getInventoryList(actorCtx, g.locationActor)
	if err != nil {
		return fmt.Errorf("error getting location inventory list: %v", err)
	}

	l, err := g.getCurrentLocation(actorCtx)
	if err != nil {
		return fmt.Errorf("error getting current location: %v", err)
	}

	g.sendUILocation(actorCtx)

	if len(inventoryList.Objects) > 0 {
		inventoryListMsg := make(indentedList, len(inventoryList.Objects)+1)
		inventoryListMsg[0] = "location inventory:"
		i := 1
		for objName, obj := range inventoryList.Objects {
			inventoryListMsg[i] = fmt.Sprintf("%s (%s)", objName, obj.Did)
			i++
		}
		g.sendUserMessage(actorCtx, inventoryListMsg)
	}

	if l.Portal != nil {
		g.sendUserMessage(actorCtx, fmt.Sprintf("you see a mysterious portal leading to %s", l.Portal.To))
	}

	g.sendArtifactHint(actorCtx, inventoryList)

	return nil
}

func (g *Game) handleCreateLocation(actorCtx actor.Context, args string) error {
	newLocation, err := g.network.CreateChainTree()
	if err != nil {
		return err
	}

	g.sendUserMessage(actorCtx, "new location created "+newLocation.MustId())
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

	result, err := actorCtx.RequestFuture(g.locationActor, &AddInteractionRequest{Interaction: interaction}, 30*time.Second).Result()
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

func (g *Game) sendUILocation(actorCtx actor.Context) {
	l, err := g.getCurrentLocation(actorCtx)

	if err != nil {
		g.sendUserMessage(actorCtx, fmt.Errorf("error getting current location: %v", err))
	}

	g.sendUserMessage(actorCtx, l)
}

func formatUserMessage(mesgInter interface{}) *jasonsgame.MessageToUser {
	msgToUser := &jasonsgame.MessageToUser{}
	switch msg := mesgInter.(type) {
	case string:
		msgToUser.Message = msg
	case []string:
		msgToUser.Message = strings.Join(msg, "\n")
	case indentedList:
		msgToUser.Message = strings.Join(msg, "\n  > ")
	case *jasonsgame.Location:
		msgToUser.Location = msg
		msgToUser.Message = msg.Description
	default:
		log.Errorf("error, unknown message type: %v", msg)
	}
	return msgToUser
}

func (g *Game) sendArtifactHint(actorCtx actor.Context, inventoryList *InventoryListResponse) {
	if inventoryList != nil && len(inventoryList.Objects) > 0 {
		for objName := range inventoryList.Objects {
			if strings.HasPrefix(objName, "artifact-") {
				g.sendUserMessage(actorCtx, hasArtifactHint)
				break
			}
		}
	}
}

func (g *Game) sendUserMessage(actorCtx actor.Context, mesgInter interface{}) {
	msgToUser := formatUserMessage(mesgInter)
	msgToUser.Sequence = g.messageSequence
	actorCtx.Send(g.ui, msgToUser)
	g.messageSequence++
}

func (g *Game) sendCommandUpdate(actorCtx actor.Context) {
	parsedCommands := make([]string, len(g.commands))
	for i, c := range g.commands {
		parsedCommands[i] = c.Parse()
	}

	cmdUpdate := &jasonsgame.CommandUpdate{Commands: parsedCommands}

	actorCtx.Send(g.ui, cmdUpdate)
}

func (g *Game) sendInvitationCommandUpdate(actorCtx actor.Context) {
	invitationCommands := []string{"help", "invitation"}

	cmdUpdate := &jasonsgame.CommandUpdate{Commands: invitationCommands}

	actorCtx.Send(g.ui, cmdUpdate)
}

func (g *Game) setLocation(actorCtx actor.Context, locationDid string) {
	oldLocationActor := g.locationActor
	if oldLocationActor != nil {
		log.Debug("found old location actor; sending stop message")
		actorCtx.Stop(g.locationActor)
	}

	log.Debug("spawning new location actor")
	g.locationActor = actorCtx.Spawn(NewLocationActorProps(&LocationActorConfig{
		Network:   g.network,
		Did:       locationDid,
		PlayerDid: g.playerTree.Did(),
	}))
	g.locationDid = locationDid

	// store previous location, except for when changing to home
	if locationDid != g.playerTree.HomeLocation.MustId() {
		err := g.ds.Put(lastLocationKey, []byte(locationDid))
		if err != nil {
			panic(errors.Wrap(err, "error saving last location"))
		}
	}

	log.Debug("replacing interactions for new location")
	err := g.replaceInteractionsFor(actorCtx, g.locationActor, oldLocationActor)
	if err != nil {
		panic(errors.Wrap(err, "error attaching interactions for location"))
	}
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
		log.Debug("deleting stale commands actor PID from cache")
		delete(g.commandsByActorCache, oldPid)
	}
	return g.refreshInteractionsFor(actorCtx, pid)
}

func (g *Game) refreshInteractionsFor(actorCtx actor.Context, pid *actor.PID) error {
	log.Debug("refreshing interactions for location actor")

	if g.commandsByActorCache == nil {
		g.commandsByActorCache = make(map[*actor.PID]commandList)
	}
	var err error

	log.Debug("updating commandsByActorCache")
	g.commandsByActorCache[pid], err = g.interactionCommandsFor(actorCtx, pid)

	if err != nil {
		log.Errorf("error updating commandsByActorCache: %+v", err)
		return err
	}

	newCommands := defaultCommandList
	for _, commands := range g.commandsByActorCache {
		newCommands = append(newCommands, commands...)
	}

	log.Debugf("setting commands to %+v", newCommands)
	g.setCommands(actorCtx, newCommands)
	return nil
}

func (g *Game) interactionCommandsFor(actorCtx actor.Context, pid *actor.PID) (commandList, error) {
	response, err := actorCtx.RequestFuture(pid, &ListInteractionsRequest{}, 30*time.Second).Result()
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
	interactionCommands := make(commandList, 0)
	for _, interactionResp := range interactions {
		switch interactionResp.Interaction.(type) {
		case *PickUpObjectInteraction:
			// Filter out PickUpObject if in player inventory
			if g.inventoryActor == pid {
				continue
			}
		case *DropObjectInteraction:
			// Filter out DropObject if NOT in player inventory
			if g.inventoryActor != pid {
				continue
			}
		}

		interactionCommands = append(interactionCommands, &interactionCommand{
			parse:       interactionResp.Interaction.GetCommand(),
			interaction: interactionResp.Interaction,
			helpGroup:   interactionResp.AttachedTo,
			did:         interactionResp.AttachedToDid,
		})
	}

	// if the location is not the players home, add portal to home command
	if pid == g.locationActor && g.locationDid != g.playerTree.HomeLocation.MustId() {
		interactionCommands = append(interactionCommands, &interactionCommand{
			parse: "portal home",
			interaction: &ChangeNamedLocationInteraction{
				Command: "portal home",
				Name:    "home",
			},
		})
	}

	return interactionCommands, nil
}

func (g *Game) getCurrentLocation(actorCtx actor.Context) (*jasonsgame.Location, error) {
	response, err := actorCtx.RequestFuture(g.locationActor, &GetLocation{}, 30*time.Second).Result()
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
