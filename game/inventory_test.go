package game

import (
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryActor_CreateObject(t *testing.T) {
	// setup

	context := actor.EmptyRootContext

	net := network.NewLocalNetwork()

	playerChainTree, err := net.CreateNamedChainTree("player")
	require.Nil(t, err)

	testPlayer := NewPlayerTree(net, playerChainTree)

	createObject, err := context.SpawnNamed(NewInventoryActorProps(&InventoryActorConfig{
		Did:     testPlayer.Did(),
		Network: net,
	}), "testCreateObject")
	require.Nil(t, err)
	defer context.Stop(createObject)

	// create first object

	response, err := context.RequestFuture(createObject, &CreateObjectRequest{Name: "test", Description: "test object"}, 1*time.Second).Result()
	require.Nil(t, err)

	createObjectResponse, ok := response.(*CreateObjectResponse)
	require.True(t, ok)
	require.Nil(t, createObjectResponse.Error)

	response, err = rootCtx.RequestFuture(createObject, &InventoryListRequest{}, 1*time.Second).Result()
	require.Nil(t, err)
	playerInventory, ok := response.(*InventoryListResponse)
	require.True(t, ok)
	require.Nil(t, playerInventory.Error)
	require.Equal(t, len(playerInventory.Objects), 1)

	netObj := NetworkObject{Object: *playerInventory.Objects["test"], Network: net}

	name, err := netObj.Name()
	require.Nil(t, err)
	assert.Equal(t, "test", name)

	desc, err := netObj.Description()
	require.Nil(t, err)
	assert.Equal(t, "test object", desc)

	// create second object

	response, err = context.RequestFuture(createObject, &CreateObjectRequest{Name: "sword", Description: "ultimate sword"}, 1*time.Second).Result()
	require.Nil(t, err)

	createObjectResponse, ok = response.(*CreateObjectResponse)
	require.True(t, ok)
	require.Nil(t, createObjectResponse.Error)

	response, err = rootCtx.RequestFuture(createObject, &InventoryListRequest{}, 1*time.Second).Result()
	require.Nil(t, err)
	playerInventory, ok = response.(*InventoryListResponse)
	require.True(t, ok)
	require.Nil(t, playerInventory.Error)
	require.Equal(t, len(playerInventory.Objects), 2)

	netObj = NetworkObject{Object: *playerInventory.Objects["sword"], Network: net}

	name, err = netObj.Name()
	require.Nil(t, err)
	assert.Equal(t, "sword", name)

	desc, err = netObj.Description()
	require.Nil(t, err)
	assert.Equal(t, "ultimate sword", desc)
}

func TestInventoryActor_Receive_NamesMustBeUnique(t *testing.T) {
	// setup
	context := actor.EmptyRootContext

	net := network.NewLocalNetwork()

	playerChainTree, err := net.CreateNamedChainTree("player")
	require.Nil(t, err)

	testPlayer := NewPlayerTree(net, playerChainTree)

	createObject, err := context.SpawnNamed(NewInventoryActorProps(&InventoryActorConfig{
		Did:     testPlayer.Did(),
		Network: net,
	}), "testCreateObject")
	require.Nil(t, err)
	defer context.Stop(createObject)

	// create first object

	response, err := context.RequestFuture(createObject, &CreateObjectRequest{Name: "test", Description: "test object"}, 1*time.Second).Result()
	require.Nil(t, err)

	createObjectResponse, ok := response.(*CreateObjectResponse)
	require.True(t, ok)
	require.Nil(t, createObjectResponse.Error)

	// try to create second object w/ same name

	response, err = context.RequestFuture(createObject, &CreateObjectRequest{Name: "test", Description: "another test"}, 1*time.Second).Result()
	require.Nil(t, err)

	createObjectResponse, ok = response.(*CreateObjectResponse)
	require.True(t, ok)
	assert.NotNil(t, createObjectResponse.Error)
	assert.Nil(t, createObjectResponse.Object)
}

func TestInventoryActor_TransferObject(t *testing.T) {
	net := network.NewLocalNetwork()

	playerChainTree, err := net.CreateNamedChainTree("player")
	require.Nil(t, err)
	testPlayer := NewPlayerTree(net, playerChainTree)

	inventory, err := rootCtx.SpawnNamed(NewInventoryActorProps(&InventoryActorConfig{
		Did:     testPlayer.Did(),
		Network: net,
	}), "testTransferObject")
	require.Nil(t, err)
	defer rootCtx.Stop(inventory)

	homeTree, err := createHome(net)
	require.Nil(t, err)
	homeActor, err := rootCtx.SpawnNamed(NewLocationActorProps(&LocationActorConfig{
		Did:     homeTree.MustId(),
		Network: net,
	}), "home")
	require.Nil(t, err)
	defer rootCtx.Stop(homeActor)

	response, err := rootCtx.RequestFuture(inventory, &CreateObjectRequest{Name: "testTransferObject", Description: "test object"}, 1*time.Second).Result()
	require.Nil(t, err)

	createObjectResponse, ok := response.(*CreateObjectResponse)
	require.True(t, ok)
	require.Nil(t, createObjectResponse.Error)

	response, err = rootCtx.RequestFuture(inventory, &TransferObjectRequest{Name: "testTransferObject", To: homeTree.MustId()}, 1*time.Second).Result()
	require.Nil(t, err)

	transferObjectResponse, ok := response.(*TransferObjectResponse)
	require.True(t, ok)
	require.Nil(t, transferObjectResponse.Error)

	response, err = rootCtx.RequestFuture(inventory, &InventoryListRequest{}, 1*time.Second).Result()
	require.Nil(t, err)
	playerInventory, ok := response.(*InventoryListResponse)
	require.True(t, ok)
	require.Nil(t, playerInventory.Error)
	require.Equal(t, len(playerInventory.Objects), 0)

	// Give time for location to pickup change and refresh
	time.Sleep(500 * time.Millisecond)

	response, err = rootCtx.RequestFuture(homeActor, &InventoryListRequest{}, 1*time.Second).Result()
	require.Nil(t, err)
	homeInventory, ok := response.(*InventoryListResponse)
	require.True(t, ok)
	require.Nil(t, homeInventory.Error)
	require.Equal(t, len(homeInventory.Objects), 1)
	require.Equal(t, homeInventory.Objects["testTransferObject"], createObjectResponse.Object)
}
