package game

import (
	"fmt"
	"strings"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"

	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"

	inventoryHandlers "github.com/quorumcontrol/jasons-game/handlers/inventory"
)

type Object struct {
	Did string
}

type InventoryActor struct {
	middleware.LogAwareHolder
	did        string
	network    network.Network
	subscriber *actor.PID
}

type InventoryActorConfig struct {
	Did     string
	Network network.Network
}

type CreateObjectRequest struct {
	Name        string
	Description string
}

type CreateObjectResponse struct {
	Object *Object
	Error  error
}

type TransferObjectRequest struct {
	Did string
	To  string
}

type TransferObjectResponse struct {
	Error error
}

type InventoryListRequest struct {
}

type InventoryListResponse struct {
	Objects map[string]*Object
	Error   error
}

func NewInventoryActorProps(cfg *InventoryActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		if cfg.Did == "" {
			panic("Must set Did in InventoryActorConfig")
		}
		if cfg.Network == nil {
			panic("Must set Network in InventoryActorConfig")
		}
		return &InventoryActor{
			did:     cfg.Did,
			network: cfg.Network,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (inv *InventoryActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		inventoryTree, err := trees.FindInventoryTree(inv.network, inv.did)
		if err != nil {
			panic(fmt.Sprintf("error finding inventory tree: %v", err))
		}
		inv.subscriber = actorCtx.Spawn(inv.network.Community().NewSubscriberProps(inventoryTree.BroadcastTopic()))
	case *CreateObjectRequest:
		inv.Log.Debugf("Received CreateObjectRequest: %+v\n", msg)
		inv.handleCreateObject(actorCtx, msg)
	case *TransferObjectRequest:
		inv.Log.Debugf("Received TransferObjectRequest: %+v\n", msg)
		inv.handleTransferObject(actorCtx, msg)
	case *InventoryListRequest:
		inv.Log.Debugf("Received InventoryListRequest: %+v\n", msg)
		inv.handleListObjects(actorCtx, msg)
	case *ListInteractionsRequest:
		inv.Log.Debugf("Received ListInteractionsRequest: %+v\n", msg)
		inv.handleListInteractionsRequest(actorCtx, msg)
	case *jasonsgame.TransferredObjectMessage:
		err := inventoryHandlers.NewUnrestrictedAddHandler(inv.network).Handle(msg)
		if err != nil {
			inv.Log.Errorf("Error on TransferredObjectMessage: %+v\n", err)
		}
	default:
		fmt.Printf("unkonwn inventory message received %v", msg)
	}
}

func (inv *InventoryActor) handleCreateObject(context actor.Context, msg *CreateObjectRequest) {
	var err error
	name := msg.Name

	object, err := trees.CreateObjectTree(inv.network, name)

	if err != nil {
		err = fmt.Errorf("error creating object chaintree: %v", err)
		inv.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}

	if msg.Description != "" {
		err := object.SetDescription(msg.Description)
		if err != nil {
			inv.Log.Warnw("error setting description of new object", "err", err)
		}
	}

	objectsPath, _ := consensus.DecodePath(trees.ObjectsPath)

	newObjectPath := strings.Join(append(objectsPath, name), "/")

	tree, err := inv.network.GetTree(inv.did)
	if err != nil {
		err = fmt.Errorf("error fetching source chaintree: %v", err)
		inv.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}

	_, err = inv.network.UpdateChainTree(tree, newObjectPath, object.MustId())
	if err != nil {
		err = fmt.Errorf("error updating objects in chaintree: %v", err)
		inv.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}

	context.Respond(&CreateObjectResponse{Object: &Object{Did: object.MustId()}})
}

func (inv *InventoryActor) handleTransferObject(context actor.Context, msg *TransferObjectRequest) {
	var err error

	objectDid := msg.Did
	if objectDid == "" {
		err = fmt.Errorf("Did is required to transfer an object")
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	if msg.To == "" {
		err = fmt.Errorf("To is required to transfer an object")
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	sourceInventory, err := trees.FindInventoryTree(inv.network, inv.did)
	if err != nil {
		err = fmt.Errorf("error fetching source chaintree: %v", err)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	exists, err := sourceInventory.Exists(objectDid)
	if err != nil {
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	if !exists {
		err = fmt.Errorf("object %v does not exist in inventory", objectDid)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	_, err = trees.FindObjectTree(inv.network, objectDid)
	if err != nil {
		err = fmt.Errorf("error fetching object chaintree %s: %v", objectDid, err)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	remoteSourceHandler, err := handlers.FindHandlerForTree(inv.network, inv.did)
	if err != nil {
		err = fmt.Errorf("error fetching handler for %v", inv.did)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}
	var sourceHandler handlers.Handler
	if remoteSourceHandler != nil {
		sourceHandler = remoteSourceHandler
	} else {
		sourceHandler = inventoryHandlers.NewUnrestrictedRemoveHandler(inv.network)
	}

	transferObjectMessage := &jasonsgame.TransferObjectMessage{
		From:   inv.did,
		To:     msg.To,
		Object: objectDid,
	}
	if !sourceHandler.Supports(transferObjectMessage) {
		err = fmt.Errorf("transfer from inventory %v is not supported", inv.did)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	remoteTargetHandler, err := handlers.FindHandlerForTree(inv.network, msg.To)
	if err != nil {
		err = fmt.Errorf("error fetching handler for %v", msg.To)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}
	if remoteTargetHandler != nil && !remoteTargetHandler.SupportsType("jasonsgame.TransferredObjectMessage") {
		err = fmt.Errorf("transfer to inventory %v is not supported", inv.did)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	if err := sourceHandler.Handle(transferObjectMessage); err != nil {
		inv.Log.Error(err)
		return
	}

	context.Respond(&TransferObjectResponse{})
}

func (inv *InventoryActor) handleListObjects(context actor.Context, msg *InventoryListRequest) {
	objects, err := inv.listObjects(context)
	if err != nil {
		context.Respond(&InventoryListResponse{Error: err})
		return
	}
	context.Respond(&InventoryListResponse{Objects: objects})
}

func (inv *InventoryActor) listObjects(context actor.Context) (map[string]*Object, error) {
	var err error

	tree, err := inv.network.GetTree(inv.did)
	if err != nil {
		err = fmt.Errorf("error fetching chaintree: %v", err)
		inv.Log.Error(err)
		return nil, err
	}

	treeObjectsPath, _ := consensus.DecodePath(fmt.Sprintf("tree/data/%s", trees.ObjectsPath))
	objectsUncasted, _, err := tree.ChainTree.Dag.Resolve(treeObjectsPath)

	if err != nil {
		err = fmt.Errorf("error fetching inventory; error: %v", err)
		inv.Log.Error(err)
		return nil, err
	}

	if objectsUncasted == nil {
		return make(map[string]*Object), nil
	}

	objects := make(map[string]*Object, len(objectsUncasted.(map[string]interface{})))
	for k, v := range objectsUncasted.(map[string]interface{}) {
		objects[k] = &Object{Did: v.(string)}
	}

	return objects, nil
}

func (inv *InventoryActor) handleListInteractionsRequest(actorCtx actor.Context, msg *ListInteractionsRequest) {
	objects, err := inv.listObjects(actorCtx)

	if err != nil {
		actorCtx.Respond(&ListInteractionsResponse{Error: err})
		return
	}

	interactions := []trees.Interaction{}

	for _, object := range objects {
		obj, err := trees.FindObjectTree(inv.network, object.Did)

		if err != nil {
			actorCtx.Respond(&ListInteractionsResponse{Error: err})
			return
		}

		objectInteractions, err := obj.InteractionsList()
		if err != nil {
			actorCtx.Respond(&ListInteractionsResponse{Error: err})
			return
		}

		interactions = append(interactions, objectInteractions...)
	}

	actorCtx.Respond(&ListInteractionsResponse{Interactions: interactions})
}

func (inv *InventoryActor) handleTransferredObjectMessage(actorCtx actor.Context, msg *jasonsgame.TransferredObjectMessage) {

}
