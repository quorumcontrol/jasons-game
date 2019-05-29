package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

const ObjectsPath = "jasons-game/player/bag-of-hodling"

type InventoryActor struct {
	middleware.LogAwareHolder

	player  *PlayerTree
	network network.Network
}

type InventoryActorConfig struct {
	Player  *PlayerTree
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

type DropObjectRequest struct {
	Name     string
	Location *jasonsgame.Location
}

type DropObjectResponse struct {
	Error error
}

type PickupObjectRequest struct {
	Name     string
	Location *jasonsgame.Location
}

type PickupObjectResponse struct {
	Object *Object
	Error  error
}

type InventoryListRequest struct {
}

type InventoryListResponse struct {
	Objects map[string]*Object
	Error   error
}

type Object struct {
	Did string
}

type NetworkObject struct {
	Object
	Network network.Network
}

func (no *NetworkObject) getProp(prop string) (string, error) {
	objectChainTree, err := no.Network.GetTree(no.Did)
	if err != nil {
		return "", err
	}

	objectNode, remainingPath, err := objectChainTree.ChainTree.Dag.Resolve([]string{"tree", "data", prop})
	if err != nil {
		return "", err
	}
	if len(remainingPath) > 0 {
		return "", fmt.Errorf("error resolving object %s: path elements remaining: %v", prop, remainingPath)
	}

	val, ok := objectNode.(string)
	if !ok {
		return "", fmt.Errorf("error casting %s to string; type is %T", prop, objectNode)
	}

	return val, nil
}

func (no *NetworkObject) ChainTree() (*consensus.SignedChainTree, error) {
	return no.Network.GetTree(no.Did)
}

func (no *NetworkObject) Name() (string, error) {
	return no.getProp("name")
}

func (no *NetworkObject) Description() (string, error) {
	return no.getProp("description")
}

func (no *NetworkObject) IsOwnedBy(keys []string) (bool, error) {
	objectChainTree, err := no.Network.GetTree(no.Did)
	if err != nil {
		return false, err
	}

	authsUncasted, remainingPath, err := objectChainTree.ChainTree.Dag.Resolve(strings.Split("tree/"+consensus.TreePathForAuthentications, "/"))
	if err != nil {
		return false, err
	}
	if len(remainingPath) > 0 {
		return false, fmt.Errorf("error resolving object: path elements remaining: %v", remainingPath)
	}

	for _, storedKey := range authsUncasted.([]interface{}) {
		found := false
		for _, checkKey := range keys {
			found = found || (storedKey.(string) == checkKey)
		}
		if !found {
			return false, nil
		}
	}

	return true, nil
}

func NewInventoryActorProps(cfg *InventoryActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &InventoryActor{
			player:  cfg.Player,
			network: cfg.Network,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (co *InventoryActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *CreateObjectRequest:
		co.Log.Debugf("Received CreateObjectRequest: %+v\n", msg)
		co.handleCreateObject(context, msg)
	case *PickupObjectRequest:
		co.Log.Debugf("Received PickupObjectRequest: %+v\n", msg)
		co.handlePickupObject(context, msg)
	case *DropObjectRequest:
		co.Log.Debugf("Received DropObjectRequest: %+v\n", msg)
		co.handleDropObject(context, msg)
	case *InventoryListRequest:
		co.Log.Debugf("Received InventoryListRequest: %+v\n", msg)
		co.handleListObjects(context, msg)
	}
}

func (co *InventoryActor) handleCreateObject(context actor.Context, msg *CreateObjectRequest) {
	var err error

	player := co.player

	if player == nil {
		err = fmt.Errorf("player is required to create an object")
		co.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}

	name := msg.Name
	chainTreeName := fmt.Sprintf("object:%s", name)

	if name == "" {
		err = fmt.Errorf("name is required to create an object")
		co.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}

	existingObj, err := co.network.GetChainTreeByName(chainTreeName)
	if err != nil {
		err = fmt.Errorf("error checking for existing chaintree; object name: %s; error: %v", name, err)
		co.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}
	if existingObj != nil {
		err = fmt.Errorf("object with name %s already exists; names must be unique", name)
		co.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}

	objectChainTree, err := co.network.CreateNamedChainTree(chainTreeName)
	if err != nil {
		err = fmt.Errorf("error creating object chaintree: %v", err)
		co.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}

	objectChainTree, err = co.network.UpdateChainTree(objectChainTree, "name", name)
	if err != nil {
		err = fmt.Errorf("error setting name of new object: %v", err)
		co.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}

	if msg.Description != "" {
		objectChainTree, err = co.network.UpdateChainTree(objectChainTree, "description", msg.Description)
		if err != nil {
			co.Log.Warnw("error setting description of new object", "err", err)
		}
	}

	playerChainTree := player.ChainTree()

	objectsPath, _ := consensus.DecodePath(ObjectsPath)

	newObjectPath := strings.Join(append(objectsPath, name), "/")

	newPlayerChainTree, err := co.network.UpdateChainTree(playerChainTree, newObjectPath, objectChainTree.MustId())

	if err != nil {
		err = fmt.Errorf("error updating objects in chaintree: %v", err)
		co.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}

	co.player.SetChainTree(newPlayerChainTree)

	context.Respond(&CreateObjectResponse{Object: &Object{Did: objectChainTree.MustId()}})
}

func (co *InventoryActor) handleDropObject(context actor.Context, msg *DropObjectRequest) {
	var err error

	player := co.player

	if player == nil {
		err = fmt.Errorf("player is required to drop an object")
		co.Log.Error(err)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	objectName := msg.Name

	if objectName == "" {
		err = fmt.Errorf("name is required to drop an object")
		co.Log.Error(err)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	if msg.Location == nil {
		err = fmt.Errorf("location is required to drop an object")
		co.Log.Error(err)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	playerChainTree := player.ChainTree()
	treeObjectsPath, _ := consensus.DecodePath(fmt.Sprintf("tree/data/%s", ObjectsPath))
	objectsUncasted, _, err := playerChainTree.ChainTree.Dag.Resolve(treeObjectsPath)
	if err != nil {
		err = fmt.Errorf("error fetching inventory; error: %v", err)
		co.Log.Error(err)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	if objectsUncasted == nil {
		err = fmt.Errorf("object %v is not in your inventory", objectName)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	objects := make(map[string]string, len(objectsUncasted.(map[string]interface{})))
	for k, v := range objectsUncasted.(map[string]interface{}) {
		objects[k] = v.(string)
	}

	if _, ok := objects[objectName]; !ok {
		err = fmt.Errorf("object %v is not in your inventory", objectName)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	locationTree, err := co.network.GetTree(msg.Location.Did)
	if err != nil {
		err = fmt.Errorf("error fetching location chaintree %s; error: %v", msg.Location.Did, err)
		co.Log.Error(err)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	locationAuthsUncasted, _, err := locationTree.ChainTree.Dag.Resolve(strings.Split("tree/"+consensus.TreePathForAuthentications, "/"))
	if err != nil {
		err = fmt.Errorf("error fetching location chaintree authentications %s; error: %v", msg.Location.Did, err)
		co.Log.Error(err)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	locationAuths := make([]string, len(locationAuthsUncasted.([]interface{})))
	for k, v := range locationAuthsUncasted.([]interface{}) {
		locationAuths[k] = v.(string)
	}

	// TODO: remove me. Checks that you can only drop in your own location. Need global broadcast first
	playerKeys, err := player.Keys()
	if err != nil {
		err = fmt.Errorf("could not fetch player keys")
		co.Log.Error(err)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	for _, storedKey := range locationAuths {
		found := false
		for _, checkKey := range playerKeys {
			found = found || storedKey == checkKey
		}
		if !found {
			err = fmt.Errorf("WIP: objects can currently only be dropped & picked up in your own land")
			context.Respond(&DropObjectResponse{Error: err})
			return
		}
	}
	// END TODO

	chainTreeName := fmt.Sprintf("object:%s", objectName)
	existingObj, err := co.network.GetChainTreeByName(chainTreeName)
	if err != nil {
		err = fmt.Errorf("error fetching object chaintree; object name: %s; error: %v", objectName, err)
		co.Log.Error(err)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	if existingObj == nil {
		err = fmt.Errorf("object %s does not exist", objectName)
		co.Log.Error(err)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	_, err = co.network.ChangeChainTreeOwner(existingObj, locationAuths)
	if err != nil {
		err = fmt.Errorf("error changing owner for object %s; error: %v", msg.Location.Did, err)
		co.Log.Error(err)
		context.Respond(&DropObjectResponse{Error: err})
		return
	}

	// TODO: switch to global topic
	co.network.Send(topicFromDid(msg.Location.Did), &jasonsgame.TransferredObjectMessage{
		From:   playerChainTree.MustId(),
		To:     msg.Location.Did,
		Object: existingObj.MustId(),
		Loc:    []int64{msg.Location.X, msg.Location.Y},
	})

	delete(objects, objectName)

	newPlayerChainTree, err := co.network.UpdateChainTree(playerChainTree, ObjectsPath, objects)
	if err != nil {
		err = fmt.Errorf("error updating objects in inventory: %v", err)
		co.Log.Error(err)
		return
	}
	co.player.SetChainTree(newPlayerChainTree)

	context.Respond(&DropObjectResponse{})
}

func (co *InventoryActor) handlePickupObject(context actor.Context, msg *PickupObjectRequest) {
	var err error

	player := co.player

	if player == nil {
		err = fmt.Errorf("player is required to pickup an object")
		co.Log.Error(err)
		context.Respond(&PickupObjectResponse{Error: err})
		return
	}

	objectName := msg.Name

	if objectName == "" {
		err = fmt.Errorf("name is required to pickup an object")
		co.Log.Error(err)
		context.Respond(&PickupObjectResponse{Error: err})
		return
	}

	if msg.Location == nil {
		err = fmt.Errorf("location is required to pickup an object")
		co.Log.Error(err)
		context.Respond(&PickupObjectResponse{Error: err})
		return
	}

	if msg.Location.Inventory == nil {
		err = fmt.Errorf("object not found")
		co.Log.Error(err)
		context.Respond(&PickupObjectResponse{Error: err})
		return
	}

	objectDid, ok := msg.Location.Inventory[objectName]

	if !ok {
		err = fmt.Errorf("object not found")
		co.Log.Error(err)
		context.Respond(&PickupObjectResponse{Error: err})
		return
	}

	playerKeys, err := player.Keys()
	if err != nil {
		err = fmt.Errorf("could not fetch player keys")
		co.Log.Error(err)
		context.Respond(&PickupObjectResponse{Error: err})
		return
	}

	// TODO: remove me. Checks that you can only pickup from your own location. Need global broadcast first
	locationTree, err := co.network.GetTree(msg.Location.Did)
	if err != nil {
		err = fmt.Errorf("error fetching location chaintree %s; error: %v", msg.Location.Did, err)
		co.Log.Error(err)
		context.Respond(&PickupObjectResponse{Error: err})
		return
	}

	locationAuthsUncasted, _, err := locationTree.ChainTree.Dag.Resolve(strings.Split("tree/"+consensus.TreePathForAuthentications, "/"))
	if err != nil {
		err = fmt.Errorf("error fetching location chaintree authentications %s; error: %v", msg.Location.Did, err)
		co.Log.Error(err)
		context.Respond(&PickupObjectResponse{Error: err})
		return
	}

	locationAuths := make([]string, len(locationAuthsUncasted.([]interface{})))
	for k, v := range locationAuthsUncasted.([]interface{}) {
		locationAuths[k] = v.(string)
	}

	for _, storedKey := range locationAuths {
		found := false
		for _, checkKey := range playerKeys {
			found = found || storedKey == checkKey
		}
		if !found {
			err = fmt.Errorf("WIP: objects can currently only be dropped & picked up in your own land")
			context.Respond(&PickupObjectResponse{Error: err})
			return
		}
	}
	// END TODO

	co.network.Send(topicFromDid(msg.Location.Did), &jasonsgame.TransferredObjectMessage{
		From:   msg.Location.Did,
		To:     player.Did(),
		Object: objectDid,
		Loc:    []int64{msg.Location.X, msg.Location.Y},
	})

	obj := NetworkObject{Object: Object{Did: objectDid}, Network: co.network}

	// TOOD: receive transfer from other land
	playerIsOwner := false
	for i := 1; i < 10; i++ {
		playerIsOwner, _ = obj.IsOwnedBy(playerKeys)

		if playerIsOwner {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	objectsPath, _ := consensus.DecodePath(ObjectsPath)
	newObjectPath := strings.Join(append(objectsPath, objectName), "/")
	newPlayerChainTree, err := co.network.UpdateChainTree(player.ChainTree(), newObjectPath, objectDid)

	if err != nil {
		err = fmt.Errorf("error updating objects in chaintree: %v", err)
		co.Log.Error(err)
		context.Respond(&PickupObjectResponse{Error: err})
		return
	}

	co.player.SetChainTree(newPlayerChainTree)
	context.Respond(&PickupObjectResponse{Object: &obj.Object})
}

func (co *InventoryActor) handleListObjects(context actor.Context, msg *InventoryListRequest) {
	var err error

	player := co.player

	if player == nil {
		err = fmt.Errorf("player is required to drop an object")
		co.Log.Error(err)
		context.Respond(&InventoryListResponse{Error: err})
		return
	}

	playerChainTree := player.ChainTree()
	treeObjectsPath, _ := consensus.DecodePath(fmt.Sprintf("tree/data/%s", ObjectsPath))
	objectsUncasted, _, err := playerChainTree.ChainTree.Dag.Resolve(treeObjectsPath)
	if err != nil {
		err = fmt.Errorf("error fetching inventory; error: %v", err)
		co.Log.Error(err)
		context.Respond(&InventoryListResponse{Error: err})
		return
	}

	if objectsUncasted == nil {
		context.Respond(&InventoryListResponse{Objects: make(map[string]*Object, 0)})
		return
	}

	objects := make(map[string]*Object, len(objectsUncasted.(map[string]interface{})))
	for k, v := range objectsUncasted.(map[string]interface{}) {
		objects[k] = &Object{Did: v.(string)}
	}

	context.Respond(&InventoryListResponse{Objects: objects})
	return
}
