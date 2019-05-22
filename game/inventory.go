package game

import (
	"fmt"
	"strings"

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

	newObject := Object{ChainTreeDID: objectChainTree.MustId()}

	newObjectPath := strings.Join(append(objectsPath, name), "/")

	newPlayerChainTree, err := co.network.UpdateChainTree(playerChainTree, newObjectPath, newObject)

	if err != nil {
		err = fmt.Errorf("error updating objects in chaintree: %v", err)
		co.Log.Error(err)
		context.Respond(&CreateObjectResponse{Error: err})
		return
	}

	co.player.SetChainTree(newPlayerChainTree)

	context.Respond(&CreateObjectResponse{Object: &newObject})
}
