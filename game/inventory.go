package game

import (
	"fmt"
	"strings"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

const ObjectsPath = "jasons-game/inventory"

const inventoryTopicSuffix = "/inventory"

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

func (inv *InventoryActor) inventoryTopicFor(did string) []byte {
	return inv.network.Community().TopicFor(did + inventoryTopicSuffix)
}

func (inv *InventoryActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		inv.subscriber = actorCtx.Spawn(inv.network.Community().NewSubscriberProps(inv.inventoryTopicFor(inv.did)))
	case *CreateObjectRequest:
		inv.Log.Debugf("Received CreateObjectRequest: %+v\n", msg)
		inv.handleCreateObject(actorCtx, msg)
	case *TransferObjectRequest:
		inv.Log.Debugf("Received TransferObjectRequest: %+v\n", msg)
		inv.handleTransferObject(actorCtx, msg)
	case *jasonsgame.TransferredObjectMessage:
		inv.Log.Debugf("Received TransferredObjectRequest: %+v\n", msg)
		inv.handleTransferredObject(actorCtx, msg)
	case *InventoryListRequest:
		inv.Log.Debugf("Received InventoryListRequest: %+v\n", msg)
		inv.handleListObjects(actorCtx, msg)
	case *ListInteractionsRequest:
		inv.handleListInteractionsRequest(actorCtx, msg)
	}
}

func (inv *InventoryActor) handleCreateObject(context actor.Context, msg *CreateObjectRequest) {
	var err error
	name := msg.Name

	object, err := CreateObjectTree(inv.network, name)

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

	objectsPath, _ := consensus.DecodePath(ObjectsPath)

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

	tree, err := inv.network.GetTree(inv.did)
	if err != nil {
		err = fmt.Errorf("error fetching source chaintree: %v", err)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	treeObjectsPath, _ := consensus.DecodePath(fmt.Sprintf("tree/data/%s", ObjectsPath))

	objectsUncasted, _, err := tree.ChainTree.Dag.Resolve(treeObjectsPath)
	if err != nil {
		err = fmt.Errorf("error fetching inventory: %v", err)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	if objectsUncasted == nil {
		err = fmt.Errorf("the inventory is empty")
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	var objectName string
	objects := make(map[string]string, len(objectsUncasted.(map[string]interface{})))
	for k, v := range objectsUncasted.(map[string]interface{}) {
		objects[k] = v.(string)

		if objectDid == objects[k] {
			objectName = k
		}
	}

	if objectName == "" {
		err = fmt.Errorf("object %v is not in the inventory", objectDid)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	targetTree, err := inv.network.GetTree(msg.To)
	if err != nil {
		err = fmt.Errorf("error fetching target chaintree: %v", err)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	targetAuthsUncasted, _, err := targetTree.ChainTree.Dag.Resolve(strings.Split("tree/"+consensus.TreePathForAuthentications, "/"))
	if err != nil {
		err = fmt.Errorf("error fetching target chaintree authentications %s; error: %v", msg.To, err)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	targetAuths := make([]string, len(targetAuthsUncasted.([]interface{})))
	for k, v := range targetAuthsUncasted.([]interface{}) {
		targetAuths[k] = v.(string)
	}

	sourceTree, err := inv.network.GetTree(inv.did)
	if err != nil {
		err = fmt.Errorf("error fetching source chaintree: %v", err)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	// TODO: remove me. Checks that you can only drop in your own location. Need global broadcast first
	sourceAuthsUncasted, _, err := sourceTree.ChainTree.Dag.Resolve(strings.Split("tree/"+consensus.TreePathForAuthentications, "/"))
	if err != nil {
		err = fmt.Errorf("error fetching source chaintree authentications %s; error: %v", msg.To, err)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	sourceAuths := make([]string, len(sourceAuthsUncasted.([]interface{})))
	for k, v := range sourceAuthsUncasted.([]interface{}) {
		sourceAuths[k] = v.(string)
	}

	for _, storedKey := range targetAuths {
		found := false
		for _, checkKey := range sourceAuths {
			found = found || storedKey == checkKey
		}
		if !found {
			err = fmt.Errorf("WIP: objects can currently only be dropped & picked up in your own land")
			context.Respond(&TransferObjectResponse{Error: err})
			return
		}
	}
	// END TODO

	existingObj, err := inv.network.GetTree(objectDid)
	if err != nil {
		err = fmt.Errorf("error fetching object chaintree %s: %v", objectDid, err)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	if existingObj == nil {
		err = fmt.Errorf("object %s does not exist", objectDid)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	existingObj, err = inv.network.ChangeChainTreeOwner(existingObj, targetAuths)
	if err != nil {
		err = fmt.Errorf("error changing owner for object %s; error: %v", objectDid, err)
		inv.Log.Error(err)
		context.Respond(&TransferObjectResponse{Error: err})
		return
	}

	if err := inv.network.Community().Send(inv.inventoryTopicFor(targetTree.MustId()), &jasonsgame.TransferredObjectMessage{
		From:   sourceTree.MustId(),
		To:     targetTree.MustId(),
		Object: existingObj.MustId(),
	}); err != nil {
		inv.Log.Error(err)
		return
	}

	delete(objects, objectName)

	_, err = inv.network.UpdateChainTree(sourceTree, ObjectsPath, objects)
	if err != nil {
		err = fmt.Errorf("error updating objects in inventory: %v", err)
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

	treeObjectsPath, _ := consensus.DecodePath(fmt.Sprintf("tree/data/%s", ObjectsPath))
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

func (inv *InventoryActor) handleTransferredObject(context actor.Context, msg *jasonsgame.TransferredObjectMessage) {
	objDid := msg.Object

	obj, err := FindObjectTree(inv.network, objDid)
	if err != nil {
		panic(fmt.Errorf("error fetching object %v: %v", objDid, err))
	}

	objName, err := obj.GetName()
	if err != nil {
		panic(fmt.Errorf("error fetching object name %v: %v", objDid, err))
	}

	tree, err := inv.network.GetTree(inv.did)
	if err != nil {
		panic(fmt.Errorf("error fetching source chaintree: %v", err))
	}

	treeObjectsPath, _ := consensus.DecodePath(fmt.Sprintf("tree/data/%s", ObjectsPath))
	objectsUncasted, _, err := tree.ChainTree.Dag.Resolve(treeObjectsPath)
	if err != nil {
		panic(fmt.Errorf("error fetching inventory: %v", err))
	}

	if objectsUncasted == nil {
		objectsUncasted = make(map[string]interface{})
	}

	objects := make(map[string]string, len(objectsUncasted.(map[string]interface{})))
	for k, v := range objectsUncasted.(map[string]interface{}) {
		objects[k] = v.(string)
	}

	if _, ok := objects[objName]; ok {
		panic(fmt.Errorf("object with %v already exists in inventory", objName))
	}

	objects[objName] = objDid

	_, err = inv.network.UpdateChainTree(tree, ObjectsPath, objects)
	if err != nil {
		panic(fmt.Errorf("error updating objects in chaintree: %v", err))
	}
}

func (inv *InventoryActor) handleListInteractionsRequest(actorCtx actor.Context, msg *ListInteractionsRequest) {
	objects, err := inv.listObjects(actorCtx)

	if err != nil {
		actorCtx.Respond(&ListInteractionsResponse{Error: err})
		return
	}

	interactions := []Interaction{}

	for _, object := range objects {
		obj, err := FindObjectTree(inv.network, object.Did)

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
