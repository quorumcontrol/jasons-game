// +build integration

package autumn

import (
	"context"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/stretchr/testify/require"
)

func TestAutumnCourt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := network.SetupTupeloNotaryGroup(ctx, true)
	require.Nil(t, err)

	net, err := network.NewRemoteNetwork(ctx, group, config.MemoryDataStore())
	require.Nil(t, err)

	courtNet, err := network.NewRemoteNetwork(ctx, group, config.MemoryDataStore())
	require.Nil(t, err)

	court := New(ctx, courtNet, "../yml-test")
	court.Start()

	playerChain, err := net.CreateLocalChainTree("player")
	require.Nil(t, err)
	playerTree, err := game.CreatePlayerTree(net, playerChain.MustId())
	require.Nil(t, err)

	rootCtx := actor.EmptyRootContext
	stream := ui.NewTestStream(t)
	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream), t.Name()+"-ui")
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
	require.Len(t, locationIds, 6)

	cmdsAndResponses := []string{
		"build portal to " + locationIds["starting"], "built a portal",
		"go through portal", "test autumn court hub",
		"visit mine 100", "you are in mine 100",
		"extract element", "element-64 has been created",
		"go back", "test autumn court hub",
		"visit mine 200", "you are in mine 200",
		"extract element", "element-c8 has been created",
		"go back", "test autumn court hub",
		"visit weaver", "you are at the weaver",
		"drop element-64", "object has been dropped",
		"drop element-c8", "object has been dropped",
		"submit offering", combinationSuccessMsg,
		"look in bag", "element-12c",
		"go back", "test autumn court hub",
		"visit mine 100", "you are in mine 100",
		"extract element", "element-64 has been created",
		"go back", "test autumn court hub",
		"visit mine 200", "you are in mine 200",
		"extract element", "element-c8 has been created",
		"go back", "test autumn court hub",
		"visit binder", "you are at the binder",
		"drop element-64", "object has been dropped",
		"drop element-c8", "object has been dropped",
		"drop element-12c", "object has been dropped",
		"submit offering", combinationSuccessMsg,
		"look in bag", "element-258",
		"go back", "test autumn court hub",
		"pick up spawn-obj", "test won",
		"look in bag", "test-autumn-prize",
	}

	for i := 0; i < len(cmdsAndResponses); i = i + 2 {
		cmd := cmdsAndResponses[i]
		response := cmdsAndResponses[i+1]
		stream.ExpectMessage(response, 60*time.Second)
		rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: cmd})
		stream.Wait()
	}
}
