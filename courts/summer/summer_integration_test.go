// +build integration

package summer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/stretchr/testify/require"
)

func TestSummerCourt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := network.SetupTupeloNotaryGroup(ctx, true)
	require.Nil(t, err)

	net, err := network.NewRemoteNetwork(ctx, group, config.MemoryDataStore())
	require.Nil(t, err)

	court := New(ctx, net, "../yml-test")
	court.Start()

	playerChain, err := net.CreateLocalChainTree("player")
	require.Nil(t, err)
	playerTree, err := game.CreatePlayerTree(net, playerChain.MustId())
	require.Nil(t, err)

	rootCtx := actor.EmptyRootContext
	stream := ui.NewTestStream(t)
	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), t.Name()+"-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(simulatedUI)

	gameCfg := &game.GameConfig{PlayerTree: playerTree, UiActor: simulatedUI, Network: net}
	gamePid, err := rootCtx.SpawnNamed(game.NewGameProps(gameCfg), t.Name()+"-game")
	require.Nil(t, err)
	defer rootCtx.Stop(gamePid)

	time.Sleep(10 * time.Second)

	ids, err := court.ids()
	require.Nil(t, err)
	require.NotNil(t, ids)
	locationIds := make(map[string]string)
	for name, did := range ids["Locations"].(map[string]interface{}) {
		locationIds[name] = did.(string)
	}
	require.Len(t, locationIds, 3)

	stream.ExpectMessage("built a portal", 3*time.Second)
	rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: fmt.Sprintf("build portal to %s", locationIds["loc1"])})
	stream.Wait()
	stream.ExpectMessage("loc1 description", 3*time.Second)
	rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: "go through portal"})
	stream.Wait()
	require.Nil(t, stream.ClearMessages())

	playerChain = playerTree.ChainTree()

	// Check artifact respawn, should be in loc1 and loc2 only
	for i := 1;  i<=5; i++ {
		time.Sleep(2 * time.Second)

		for locationName, did := range locationIds {
			locTree, err := net.GetTree(did)
			require.Nil(t, err)
			locInventory := trees.NewInventoryTree(net, locTree)

			spawnObjecetDid, err := locInventory.DidForName("artifact-test")
			require.Nil(t, err)
			if spawnObjecetDid == "" {
				continue
			}
			if locationName == "loc3" {
				require.Fail(t, "artifact spawned in loc3, should have only spawned in loc1 / loc2")
			}

			stream.ExpectMessage(locationName + " description", 2*time.Second)
			rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: "go " + locationName})
			stream.Wait()
			time.Sleep(50 * time.Millisecond)

			rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: "refresh"})
			time.Sleep(50 * time.Millisecond)

			stream.ExpectMessage("object has been picked up", 5*time.Second)
			rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: "pick up artifact-test"})
			stream.Wait()
			time.Sleep(50 * time.Millisecond)

			stream.ExpectMessage("artifact-test", 2*time.Second)
			rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: "look in bag"})
			stream.Wait()

			// TODO: check artifact

			// empty inventory so we can pick up same object again
			playerChain, err = net.UpdateChainTree(playerChain, trees.ObjectsPath, map[string]interface{}{})
			require.Nil(t, err)
			require.Nil(t, stream.ClearMessages())
		}
	}

	// Check winning prize respawn, should be in loc3
	for i := 1;  i<=2; i++ {
		time.Sleep(2 * time.Second)

		stream.ExpectMessage("loc3 description", 2*time.Second)
		rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: "go loc3"})
		stream.Wait()
		time.Sleep(50 * time.Millisecond)

		stream.ExpectMessage("Woah did you kill it?", 2*time.Second)
		rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: "pick up spawn-obj"})
		stream.Wait()
		time.Sleep(50 * time.Millisecond)

		stream.ExpectMessage("test-summer-prize", 2*time.Second)
		rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: "look in bag"})
		stream.Wait()

		// Check to make sure the object that the user requested to pickup
		// got transformed into the prize with the right data
		playerInventory, err := trees.FindInventoryTree(net, playerChain.MustId())
		require.Nil(t, err)
		objDid, err := playerInventory.DidForName("test-summer-prize")
		require.Nil(t, err)
		objectTree, err := game.FindObjectTree(net, objDid)
		require.Nil(t, err)
		desc, err := objectTree.GetDescription()
		require.Nil(t, err)
		require.Equal(t, desc, "the test won the summer court")
		interactions, err := objectTree.InteractionsList()
		require.Nil(t, err)
		require.Len(t, interactions, 0)

		// empty inventory so we can pick up same object again
		playerChain, err = net.UpdateChainTree(playerChain, trees.ObjectsPath, map[string]interface{}{})
		require.Nil(t, err)
		require.Nil(t, stream.ClearMessages())
	}
}