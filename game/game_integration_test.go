// +build integration

package game

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
)

func TestFullIntegration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := network.SetupTupeloNotaryGroup(ctx, true)
	require.Nil(t, err)

	path := "/tmp/test-full-game"

	err = os.RemoveAll(path)
	require.Nil(t, err)

	err = os.MkdirAll(path, 0755)
	require.Nil(t, err)

	defer os.RemoveAll(path)

	ds, err := config.LocalDataStore(path)
	require.Nil(t, err)

	net, err := network.NewRemoteNetwork(ctx, group, ds)
	require.Nil(t, err)

	rootCtx := actor.EmptyRootContext

	stream := ui.NewTestStream()

	uiActor, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), "test-integration-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(uiActor)

	playerChain, err := net.CreateNamedChainTree("player")
	require.Nil(t, err)
	playerTree, err := CreatePlayerTree(net, playerChain.MustId())
	require.Nil(t, err)

	gameCfg := &GameConfig{PlayerTree: playerTree, UiActor: uiActor, Network: net}
	gameActor, err := rootCtx.SpawnNamed(NewGameProps(gameCfg),
		"test-integration-game")
	require.Nil(t, err)
	defer rootCtx.Stop(gameActor)

	readyFut := rootCtx.RequestFuture(gameActor, &ping{}, 15*time.Second)
	// wait on the game actor being ready
	_, err = readyFut.Result()
	require.Nil(t, err)

	someTree, err := net.CreateChainTree()
	require.Nil(t, err)
	locationTree := NewLocationTree(net, someTree)
	err = locationTree.AddInteraction(&RespondInteraction{
		Command:  "atesthiddencommand",
		Response: "hello",
		Hidden:   true,
	})
	require.Nil(t, err)

	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: fmt.Sprintf("connect location %s as enter dungeon", someTree.MustId())})
	time.Sleep(100 * time.Millisecond)
	msgs := filterUserMessages(t, stream.GetMessages())
	require.Len(t, msgs, 2)

	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "enter dungeon"})
	time.Sleep(100 * time.Millisecond)
	msgs = filterUserMessages(t, stream.GetMessages())
	require.Len(t, msgs, 3)

	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "atesthiddencommand"})
	time.Sleep(100 * time.Millisecond)
	require.Len(t, msgs, 4)
	require.Equal(t, msgs[3].Message, "hello")

	err = stream.ClearMessages()
	require.Nil(t, err)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "help"})
	time.Sleep(100 * time.Millisecond)
	msgs = filterUserMessages(t, stream.GetMessages())
	includesHidden := false
	for _, msg := range msgs {
		if msg.Message == "atesthiddencommand" {
			includesHidden = true
		}
	}
	require.False(t, includesHidden)
}
