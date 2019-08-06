package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"

	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	inventoryHandlers "github.com/quorumcontrol/jasons-game/handlers/inventory"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"

	"github.com/quorumcontrol/messages/build/go/signatures"
)

type InventoryActor struct {
	middleware.LogAwareHolder
	did        string
	network    network.Network
	subscriber *actor.PID
	handler    handlers.Handler
}

type InventoryActorConfig struct {
	Did     string
	Network network.Network
	Handler handlers.Handler
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
			handler: cfg.Handler,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (inv *InventoryActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		inv.initialize(actorCtx)
	case *CreateObjectRequest:
		inv.Log.Debugf("Received CreateObjectRequest: %+v\n", msg)
		inv.handleCreateObject(actorCtx, msg)
	case *TransferObjectRequest:
		inv.Log.Debugf("Received TransferObjectRequest: %+v\n", msg)
		inv.handleTransferObject(actorCtx, msg)
	case *jasonsgame.TransferredObjectMessage:
		inv.Log.Debugf("Received TransferredObjectRequest: %+v\n", msg)
		err := inv.handler.Handle(msg)
		if err != nil {
			inv.Log.Errorf("Error on TransferredObjectMessage: %+v\n", err)
		}
	case *InventoryListRequest:
		inv.Log.Debugf("Received InventoryListRequest: %+v\n", msg)
		inv.handleListObjects(actorCtx, msg)
	case *ListInteractionsRequest:
		inv.handleListInteractionsRequest(actorCtx, msg)
	case *signatures.CurrentState:
		if parentPID := actorCtx.Parent(); parentPID != nil {
			actorCtx.Send(parentPID, &StateChange{PID: actorCtx.Self()})
		}
	}
}

func (inv *InventoryActor) initialize(actorCtx actor.Context) {
	inventoryTree, err := trees.FindInventoryTree(inv.network, inv.did)
	if err != nil {
		panic(fmt.Sprintf("error finding inventory tree: %v", err))
	}

	actorCtx.Spawn(inv.network.NewCurrentStateSubscriptionProps(inv.did))

	inv.subscriber = actorCtx.Spawn(inv.network.Community().NewSubscriberProps(inventoryTree.BroadcastTopic()))

	if inv.handler == nil {
		inv.handler = inv.pickDefaultHandler(actorCtx, inventoryTree)
	}
}

func (inv *InventoryActor) pickDefaultHandler(actorCtx actor.Context, inventoryTree *trees.InventoryTree) handlers.Handler {
	chaintreeHandler, err := handlers.FindHandlerForTree(inv.network, inventoryTree.MustId())
	if err != nil {
		panic(fmt.Sprintf("error finding handler for inventory: %v", err))
	}
	if chaintreeHandler != nil {
		return chaintreeHandler
	}

	localKeyAddr := consensus.DidToAddr(consensus.EcdsaPubkeyToDid(*inv.network.PublicKey()))
	isLocal, err := inventoryTree.IsOwnedBy([]string{localKeyAddr})
	if err != nil {
		panic(fmt.Sprintf("error check owner for inventory: %v", err))
	}

	if isLocal {
		return handlers.NewCompositeHandler([]handlers.Handler{
			inventoryHandlers.NewUnrestrictedAddHandler(inv.network),
			inventoryHandlers.NewUnrestrictedRemoveHandler(inv.network),
		})
	} else {
		return handlers.NewNoopHandler()
	}
}

func (inv *InventoryActor) handleCreateObject(actorCtx actor.Context, msg *CreateObjectRequest) {
	var err error
	name := msg.Name

	object, err := CreateObjectTree(inv.network, name)

	if err != nil {
		err = fmt.Errorf("error creating object chaintree: %v", err)
		inv.Log.Error(err)
		actorCtx.Respond(&CreateObjectResponse{Error: err})
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
		actorCtx.Respond(&CreateObjectResponse{Error: err})
		return
	}

	_, err = inv.network.UpdateChainTree(tree, newObjectPath, object.MustId())
	if err != nil {
		err = fmt.Errorf("error updating objects in chaintree: %v", err)
		inv.Log.Error(err)
		actorCtx.Respond(&CreateObjectResponse{Error: err})
		return
	}

	actorCtx.Respond(&CreateObjectResponse{Object: &Object{Did: object.MustId()}})
}

func (inv *InventoryActor) handleTransferObject(actorCtx actor.Context, msg *TransferObjectRequest) {
	var err error

	objectDid := msg.Did
	if objectDid == "" {
		err = fmt.Errorf("Did is required to transfer an object")
		inv.Log.Error(err)
		actorCtx.Respond(&TransferObjectResponse{Error: err})
		return
	}

	if msg.To == "" {
		err = fmt.Errorf("To is required to transfer an object")
		inv.Log.Error(err)
		actorCtx.Respond(&TransferObjectResponse{Error: err})
		return
	}

	sourceInventory, err := trees.FindInventoryTree(inv.network, inv.did)
	if err != nil {
		err = fmt.Errorf("error fetching source chaintree: %v", err)
		inv.Log.Error(err)
		actorCtx.Respond(&TransferObjectResponse{Error: err})
		return
	}

	exists, err := sourceInventory.Exists(objectDid)
	if err != nil {
		inv.Log.Error(err)
		actorCtx.Respond(&TransferObjectResponse{Error: err})
		return
	}

	if !exists {
		err = fmt.Errorf("object %v does not exist in inventory", objectDid)
		inv.Log.Error(err)
		actorCtx.Respond(&TransferObjectResponse{Error: err})
		return
	}

	_, err = FindObjectTree(inv.network, objectDid)
	if err != nil {
		err = fmt.Errorf("error fetching object chaintree %s: %v", objectDid, err)
		inv.Log.Error(err)
		actorCtx.Respond(&TransferObjectResponse{Error: err})
		return
	}

	transferObjectMessage := &jasonsgame.RequestObjectTransferMessage{
		From:   inv.did,
		To:     msg.To,
		Object: objectDid,
	}
	if !inv.handler.Supports(transferObjectMessage) {
		err = fmt.Errorf("transfer from inventory %v is not supported", inv.did)
		inv.Log.Error(err)
		actorCtx.Respond(&TransferObjectResponse{Error: err})
		return
	}

	remoteTargetHandler, err := handlers.FindHandlerForTree(inv.network, msg.To)
	if err != nil {
		err = fmt.Errorf("error fetching handler for %v", msg.To)
		inv.Log.Error(err)
		actorCtx.Respond(&TransferObjectResponse{Error: err})
		return
	}
	if remoteTargetHandler != nil && !remoteTargetHandler.Supports((*jasonsgame.TransferredObjectMessage)(nil)) {
		err = fmt.Errorf("transfer to inventory %v is not supported", inv.did)
		inv.Log.Error(err)
		actorCtx.Respond(&TransferObjectResponse{Error: err})
		return
	}

	if err := inv.handler.Handle(transferObjectMessage); err != nil {
		inv.Log.Error(err)
		return
	}

	actorCtx.Respond(&TransferObjectResponse{})
}

func (inv *InventoryActor) handleListObjects(actorCtx actor.Context, msg *InventoryListRequest) {
	objects, err := inv.listObjects(actorCtx)
	if err != nil {
		actorCtx.Respond(&InventoryListResponse{Error: err})
		return
	}
	actorCtx.Respond(&InventoryListResponse{Objects: objects})
}

func (inv *InventoryActor) listObjects(actorCtx actor.Context) (map[string]*Object, error) {
	var err error
	ctx := context.TODO()

	tree, err := inv.network.GetTree(inv.did)
	if err != nil {
		err = fmt.Errorf("error fetching chaintree: %v", err)
		inv.Log.Error(err)
		return nil, err
	}

	treeObjectsPath, _ := consensus.DecodePath(fmt.Sprintf("tree/data/%s", trees.ObjectsPath))
	objectsUncasted, _, err := tree.ChainTree.Dag.Resolve(ctx, treeObjectsPath)

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

	interactions := []*InteractionResponse{}
	if len(objects) == 0 {
		actorCtx.Respond(&ListInteractionsResponse{Interactions: interactions})
		return
	}

	interactionsCh := make(chan []*InteractionResponse, len(objects))
	errCh := make(chan error)

	for _, obj := range objects {
		go func(object *Object) {
			obj, err := FindObjectTree(inv.network, object.Did)
			if err != nil {
				errCh <- err
				return
			}

			name, err := obj.GetName()
			if err != nil {
				errCh <- err
				return
			}

			objectInteractions, err := obj.InteractionsList()
			if err != nil {
				errCh <- err
				return
			}

			interactionResp := make([]*InteractionResponse, len(objectInteractions))
			for i, interaction := range objectInteractions {
				interactionResp[i] = &InteractionResponse{
					AttachedTo:    name,
					AttachedToDid: object.Did,
					Interaction:   interaction,
				}
			}

			interactionsCh <- interactionResp
		}(obj)
	}

	receivedCount := 0
	done := false

	for !done {
		select {
		case err := <-errCh:
			actorCtx.Respond(&ListInteractionsResponse{Error: err})
			return
		case objectInteractions := <-interactionsCh:
			interactions = append(interactions, objectInteractions...)
			receivedCount++
			done = receivedCount >= len(objects)
		}
	}

	actorCtx.Respond(&ListInteractionsResponse{Interactions: interactions})
}
