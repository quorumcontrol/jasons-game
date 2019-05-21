package game

import (
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
)

func TestCreateObjectActor_Receive(t *testing.T) {
	// setup

	context := actor.EmptyRootContext

	net := network.NewLocalNetwork()

	createObject, err := context.SpawnNamed(NewCreateObjectActorProps(&CreateObjectActorConfig{
		Network: net,
	}), "testCreateObject")
	require.Nil(t, err)

	playerChainTree, err := net.CreateNamedChainTree("player")
	require.Nil(t, err)

	testPlayer := game.NewPlayer(playerChainTree)

	// create first object

	context.Send(createObject, &CreateObjectMessage{Player: testPlayer, Name: "test", Description: "test object"})

	time.Sleep(100 * time.Millisecond)

	objectsPath, _ := consensus.DecodePath(ObjectsPath)
	playerObjectsNode, remainingPath, err := testPlayer.ChainTree().ChainTree.Dag.Resolve(append([]string{"tree", "data"}, objectsPath...))
	require.Nil(t, err)
	require.Empty(t, remainingPath)

	playerObjects := make([]Object, 0)

	err = typecaster.ToType(playerObjectsNode, &playerObjects)
	require.Nil(t, err)

	assert.Len(t, playerObjects, 1)

	objectChainTree, err := net.GetChainTreeByName("object:test")
	require.Nil(t, err)

	assert.Equal(t, Object{ChainTreeDID: objectChainTree.MustId()}, playerObjects[0])

	// create second object

	context.Send(createObject, &CreateObjectMessage{Player: testPlayer, Name: "sword", Description: "ultimate sword"})

	time.Sleep(100 * time.Millisecond)

	playerObjectsNode, remainingPath, err = testPlayer.ChainTree().ChainTree.Dag.Resolve(append([]string{"tree", "data"}, objectsPath...))
	require.Nil(t, err)
	require.Empty(t, remainingPath)

	playerObjects = make([]Object, 0)

	err = typecaster.ToType(playerObjectsNode, &playerObjects)
	require.Nil(t, err)

	assert.Len(t, playerObjects, 2)

	objectChainTree, err = net.GetChainTreeByName("object:sword")
	require.Nil(t, err)

	assert.Equal(t, Object{ChainTreeDID: objectChainTree.MustId()}, playerObjects[1])
}
