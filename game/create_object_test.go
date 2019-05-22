package game

import (
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/jasons-game/network"
)

func TestCreateObjectActor_Receive(t *testing.T) {
	// setup

	context := actor.EmptyRootContext

	net := network.NewLocalNetwork()

	playerChainTree, err := net.CreateNamedChainTree("player")
	require.Nil(t, err)

	testPlayer := NewPlayer(playerChainTree)

	createObject, err := context.SpawnNamed(NewCreateObjectActorProps(&CreateObjectActorConfig{
		Player:  testPlayer,
		Network: net,
	}), "testCreateObject")
	require.Nil(t, err)
	defer context.Stop(createObject)

	// create first object

	response, err := context.RequestFuture(createObject, &CreateObjectRequest{Name: "test", Description: "test object"}, 1 * time.Second).Result()
	require.Nil(t, err)

	createObjectResponse, ok := response.(*CreateObjectResponse)
	require.True(t, ok)
	require.Nil(t, createObjectResponse.Error)

	objectsPath, _ := consensus.DecodePath(ObjectsPath)
	objectPath := append(objectsPath, "test")
	playerObjNode, remainingPath, err := testPlayer.ChainTree().ChainTree.Dag.Resolve(append([]string{"tree", "data"}, objectPath...))
	require.Nil(t, err)
	require.Empty(t, remainingPath)

	playerObj := Object{}

	err = typecaster.ToType(playerObjNode, &playerObj)
	require.Nil(t, err)

	objectChainTree, err := net.GetChainTreeByName("object:test")
	require.Nil(t, err)

	obj := Object{ChainTreeDID: objectChainTree.MustId()}
	assert.Equal(t, obj, playerObj)

	assert.Equal(t, obj.ChainTreeDID, createObjectResponse.Object.ChainTreeDID)

	netObj := NetworkObject{Object: obj, Network: net}

	name, err := netObj.Name()
	require.Nil(t, err)
	assert.Equal(t, "test", name)

	desc, err := netObj.Description()
	require.Nil(t, err)
	assert.Equal(t, "test object", desc)

	// create second object

	response, err = context.RequestFuture(createObject, &CreateObjectRequest{Name: "sword", Description: "ultimate sword"}, 1 * time.Second).Result()
	require.Nil(t, err)

	createObjectResponse, ok = response.(*CreateObjectResponse)
	require.True(t, ok)
	require.Nil(t, createObjectResponse.Error)

	objectPath = append(objectsPath, "sword")
	playerObjNode, remainingPath, err = testPlayer.ChainTree().ChainTree.Dag.Resolve(append([]string{"tree", "data"}, objectPath...))
	require.Nil(t, err)
	require.Empty(t, remainingPath)

	playerObj = Object{}

	err = typecaster.ToType(playerObjNode, &playerObj)
	require.Nil(t, err)

	objectChainTree, err = net.GetChainTreeByName("object:sword")
	require.Nil(t, err)

	obj = Object{ChainTreeDID: objectChainTree.MustId()}
	assert.Equal(t, obj, playerObj)

	assert.Equal(t, obj.ChainTreeDID, createObjectResponse.Object.ChainTreeDID)

	netObj = NetworkObject{Object: obj, Network: net}

	name, err = netObj.Name()
	require.Nil(t, err)
	assert.Equal(t, "sword", name)

	desc, err = netObj.Description()
	require.Nil(t, err)
	assert.Equal(t, "ultimate sword", desc)
}

func TestCreateObjectActor_Receive_NamesMustBeUnique(t *testing.T) {
	// setup

	context := actor.EmptyRootContext

	net := network.NewLocalNetwork()

	playerChainTree, err := net.CreateNamedChainTree("player")
	require.Nil(t, err)

	testPlayer := NewPlayer(playerChainTree)

	createObject, err := context.SpawnNamed(NewCreateObjectActorProps(&CreateObjectActorConfig{
		Player:  testPlayer,
		Network: net,
	}), "testCreateObject")
	require.Nil(t, err)
	defer context.Stop(createObject)

	// create first object

	response, err := context.RequestFuture(createObject, &CreateObjectRequest{Name: "test", Description: "test object"}, 1 * time.Second).Result()
	require.Nil(t, err)

	createObjectResponse, ok := response.(*CreateObjectResponse)
	require.True(t, ok)
	require.Nil(t, createObjectResponse.Error)

	// try to create second object w/ same name

	response, err = context.RequestFuture(createObject, &CreateObjectRequest{Name: "test", Description: "another test"}, 1 * time.Second).Result()
	require.Nil(t, err)

	createObjectResponse, ok = response.(*CreateObjectResponse)
	require.True(t, ok)
	assert.NotNil(t, createObjectResponse.Error)
	assert.Nil(t, createObjectResponse.Object)
}
