package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"

	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	inventoryHandlers "github.com/quorumcontrol/jasons-game/handlers/inventory"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"

	"github.com/quorumcontrol/messages/build/go/signatures"
)

var ErrExists = errors.New("inventory: object already exists")

type InventoryActor struct {
	middleware.LogAwareHolder
	did        string
	inventory  *trees.InventoryTree
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
	Name             string
	Description      string
	WithInscriptions bool
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
		if string(msg.Signature.ObjectId) == inv.did {
			newTip, err := cid.Cast(msg.Signature.NewTip)
			if err != nil {
				inv.Log.Error(errors.Wrap(err, "could not refresh inventory"))
				return
			}
			refreshedInventory, err := inv.network.GetTreeByTip(newTip)
			if err != nil {
				inv.Log.Error(errors.Wrap(err, "could not refresh inventory"))
				return
			}
			err = inv.network.TreeStore().UpdateTreeMetadata(refreshedInventory)
			if err != nil {
				inv.Log.Error(errors.Wrap(err, "could not refresh inventory"))
				return
			}
			inv.inventory = trees.NewInventoryTree(inv.network, refreshedInventory)
		}

		if parentPID := actorCtx.Parent(); parentPID != nil {
			actorCtx.Send(parentPID, &StateChange{PID: actorCtx.Self()})
		}
	}
}

func (inv *InventoryActor) initialize(actorCtx actor.Context) {
	var err error
	inv.inventory, err = trees.FindInventoryTree(inv.network, inv.did)
	if err != nil {
		panic(fmt.Sprintf("error finding inventory tree: %v", err))
	}

	actorCtx.Spawn(inv.network.NewCurrentStateSubscriptionProps(inv.did))

	inv.subscriber = actorCtx.Spawn(inv.network.Community().NewSubscriberProps(inv.inventory.BroadcastTopic()))

	if inv.handler == nil {
		inv.handler = inv.pickDefaultHandler(actorCtx)
	}
}

func (inv *InventoryActor) pickDefaultHandler(actorCtx actor.Context) handlers.Handler {
	chaintreeHandler, err := handlers.FindHandlerForTree(inv.network, inv.inventory.MustId())
	if err != nil {
		panic(fmt.Sprintf("error finding handler for inventory: %v", err))
	}
	if chaintreeHandler != nil {
		return chaintreeHandler
	}

	localKeyAddr := consensus.DidToAddr(consensus.EcdsaPubkeyToDid(*inv.network.PublicKey()))
	isLocal, err := inv.inventory.IsOwnedBy([]string{localKeyAddr})
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

	exists, err := inv.inventory.DidForName(name)
	if err != nil {
		err = fmt.Errorf("error checking inventory chaintree: %v", err)
		inv.Log.Error(err)
		actorCtx.Respond(&CreateObjectResponse{Error: err})
		return
	}

	if len(exists) > 0 {
		actorCtx.Respond(&CreateObjectResponse{
			Error: ErrExists,
		})
		return
	}

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
			err = fmt.Errorf("error setting description of new object: %v", err)
			inv.Log.Error(err)
			actorCtx.Respond(&CreateObjectResponse{Error: err})
			return
		}
	}

	if msg.WithInscriptions {
		err := object.AddDefaultInscriptionInteractions()
		if err != nil {
			err = fmt.Errorf("error adding inscription commands: %v", err)
			inv.Log.Error(err)
			actorCtx.Respond(&CreateObjectResponse{Error: err})
			return
		}
	}

	err = inv.inventory.Add(object.MustId())
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

	exists, err := inv.inventory.Exists(objectDid)
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

	all, err := inv.inventory.All()
	if err != nil {
		err = fmt.Errorf("error fetching inventory; error: %v", err)
		inv.Log.Error(err)
		return nil, err
	}

	objects := make(map[string]*Object, len(all))
	for did, name := range all {
		objects[name] = &Object{Did: did}
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
