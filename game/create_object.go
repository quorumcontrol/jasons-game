package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"

	"github.com/quorumcontrol/jasons-game/network"
)

const ObjectsPath = "jasons-game/player/bag-of-hodling"

func init() {
	cbornode.RegisterCborType(Object{})
	typecaster.AddType(Object{})
}

type CreateObjectActor struct {
	middleware.LogAwareHolder

	player  *Player
	network network.Network
}

type CreateObjectActorConfig struct {
	Player  *Player
	Network network.Network
}

type CreateObjectRequest struct {
	Name        string
	Description string
}

type CreateObjectResponse struct {
	Object Object
}

type Object struct {
	ChainTreeDID string
}

type NetworkObject struct {
	Object
	Network network.Network
}

func (no *NetworkObject) getProp(prop string) (string, error) {
	objectChainTree, err := no.Network.GetTree(no.ChainTreeDID)
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

func (no *NetworkObject) Name() (string, error) {
	return no.getProp("name")
}

func (no *NetworkObject) Description() (string, error) {
	return no.getProp("description")
}

func NewCreateObjectActorProps(cfg *CreateObjectActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &CreateObjectActor{
			player:  cfg.Player,
			network: cfg.Network,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (co *CreateObjectActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *CreateObjectRequest:
		co.Log.Debugf("Received CreateObjectRequest: %+v\n", msg)
		co.handleCreateObject(context, msg)
	}
}

func (co *CreateObjectActor) handleCreateObject(context actor.Context, msg *CreateObjectRequest) {
	player := co.player

	if player == nil {
		co.Log.Error("player is required to create an object")
		return
	}

	chainTreeName := fmt.Sprintf("object:%s", msg.Name)
	objectChainTree, err := co.network.CreateNamedChainTree(chainTreeName)
	if err != nil {
		co.Log.Errorw("error creating object chaintree", "err", err)
		return
	}

	objectChainTree, err = co.network.UpdateChainTree(objectChainTree, "name", msg.Name)
	if err != nil {
		co.Log.Errorw("error setting name of new object", "err", err)
		return
	}

	objectChainTree, err = co.network.UpdateChainTree(objectChainTree, "description", msg.Description)
	if err != nil {
		co.Log.Errorw("error setting description of new object", "err", err)
		return
	}

	playerChainTree := player.ChainTree()

	objectsPath, _ := consensus.DecodePath(ObjectsPath)
	existingObjectsNode, remainingPath, err := playerChainTree.ChainTree.Dag.Resolve(append([]string{"tree", "data"}, objectsPath...))
	if err != nil {
		co.Log.Errorw("error resolving existing player objects", "err", err)
		return
	}

	var newObject Object
	var newObjects map[string]Object

	if len(remainingPath) > 0 {
		newObject = Object{ChainTreeDID: objectChainTree.MustId()}
		newObjects = map[string]Object{msg.Name: newObject}
	} else {
		existingObjects := make(map[string]Object)

		err = typecaster.ToType(existingObjectsNode, &existingObjects)
		if err != nil {
			co.Log.Errorw("error casting existing objects to string slice", "err", err)
			return
		}

		newObject = Object{ChainTreeDID: objectChainTree.MustId()}
		newObjects = existingObjects
		newObjects[msg.Name] = newObject
	}

	newPlayerChainTree, err := co.network.UpdateChainTree(playerChainTree, ObjectsPath, newObjects)

	if err != nil {
		co.Log.Errorw("error updating objects in chaintree", "err", err)
		return
	}

	co.player.SetChainTree(newPlayerChainTree)

	context.Respond(&CreateObjectResponse{Object: newObject})
}
