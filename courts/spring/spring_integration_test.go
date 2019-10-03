// +build integration

package spring

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

func setupTestSpringCourt(t *testing.T, ctx context.Context) (*ui.TestStream, *actor.PID) {
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
	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream), t.Name()+"-ui")
	require.Nil(t, err)

	gameCfg := &game.GameConfig{PlayerTree: playerTree, UiActor: simulatedUI, Network: net}
	gamePid, err := rootCtx.SpawnNamed(game.NewGameProps(gameCfg), t.Name()+"-game")
	require.Nil(t, err)

	time.Sleep(10 * time.Second)

	ids, err := court.court.Ids()
	require.Nil(t, err)
	require.NotNil(t, ids)
	locationIds := make(map[string]string)
	for name, did := range ids["Locations"].(map[string]interface{}) {
		locationIds[name] = did.(string)
	}
	require.Len(t, locationIds, 5)

	stream.ExpectMessage("built a portal", 3*time.Second)
	rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: fmt.Sprintf("build portal to %s", locationIds["main"])})
	stream.Wait()
	stream.ExpectMessage("main description", 3*time.Second)
	rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: "go through portal"})
	stream.Wait()
	require.Nil(t, stream.ClearMessages())

	playerInventory := trees.NewInventoryTree(net, playerChain)
	for i := 1; i <= 3; i++ {
		obj, err := game.CreateObjectTree(net, fmt.Sprintf("page-%d", i))
		require.Nil(t, err)
		_, err = net.UpdateChainTree(obj.ChainTree(), "jasons-game/inscriptions", []string{fmt.Sprintf("inscription%d", i)})
		require.Nil(t, err)
		err = playerInventory.ForceAdd(obj.MustId())
		require.Nil(t, err)
	}

	go func() {
		<-ctx.Done()
		rootCtx.Stop(gamePid)
		rootCtx.Stop(simulatedUI)
	}()

	return stream, gamePid
}

func TestSpringCourt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, gamePid := setupTestSpringCourt(t, ctx)

	cmdsAndResponses := []string{
		"go loc1", "loc1 description",
		"drop page-1", "object has been dropped",
		"go loc2", "loc2 description",
		"drop page-2", "object has been dropped",
		"go loc3", "loc3 description",
		"drop page-3", "object has been dropped",
		"go main", "main description",
		"pick up spawn-obj", "test won",
		"look in bag", "test-spring-prize",
	}

	for i := 0; i < len(cmdsAndResponses); i = i + 2 {
		cmd := cmdsAndResponses[i]
		response := cmdsAndResponses[i+1]
		stream.ExpectMessage(response, 20*time.Second)
		actor.EmptyRootContext.Send(gamePid, &jasonsgame.UserInput{Message: cmd})
		stream.Wait()
	}
}

func TestSpringCourtFail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, gamePid := setupTestSpringCourt(t, ctx)

	cmdsAndResponses := []string{
		"go loc1", "loc1 description",
		"drop page-1", "object has been dropped",
		"go loc2", "loc2 description",
		"drop page-3", "object has been dropped",
		"go loc3", "loc3 description",
		"drop page-2", "object has been dropped",
		"go main", "main description",
		"pick up spawn-obj", "your pedestal placement is incorrect",
		"look in bag", "empty",
	}

	for i := 0; i < len(cmdsAndResponses); i = i + 2 {
		cmd := cmdsAndResponses[i]
		response := cmdsAndResponses[i+1]
		stream.ExpectMessage(response, 20*time.Second)
		actor.EmptyRootContext.Send(gamePid, &jasonsgame.UserInput{Message: cmd})
		stream.Wait()
	}
}
