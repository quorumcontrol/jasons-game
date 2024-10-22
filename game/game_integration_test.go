// +build integration

package game

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
)

func TestFullIntegration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := network.SetupTupeloNotaryGroup(ctx, true)
	require.Nil(t, err)

	net, err := network.NewRemoteNetwork(ctx, group, config.MemoryDataStore())
	require.Nil(t, err)

	rootCtx := actor.EmptyRootContext

	stream := ui.NewTestStream(t)

	uiActor, err := rootCtx.SpawnNamed(ui.NewUIProps(stream), "test-integration-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(uiActor)

	playerChain, err := net.CreateLocalChainTree("player")
	require.Nil(t, err)
	playerTree, err := CreatePlayerTree(net, playerChain.MustId())
	require.Nil(t, err)

	otherPlayerTree, err := net.CreateChainTree()
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
	err = locationTree.SetDescription("in the dungeon")
	require.Nil(t, err)
	err = locationTree.AddInteraction(&RespondInteraction{
		Command:  "atesthiddencommand",
		Response: "hello",
		Hidden:   true,
	})
	require.Nil(t, err)
	time.Sleep(300 * time.Millisecond)

	stream.ExpectMessage("added a connection", 3*time.Second)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: fmt.Sprintf("connect location %s as enter dungeon", someTree.MustId())})
	stream.Wait()

	stream.ExpectMessage("in the dungeon", 3*time.Second)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "enter dungeon"})
	stream.Wait()

	stream.ExpectMessage("hello", 3*time.Second)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "atesthiddencommand"})
	stream.Wait()

	stream.ExpectMessage("transfer-test has been created", 3*time.Second)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "create object transfer-test"})
	stream.Wait()

	inv, err := trees.FindInventoryTree(net, playerTree.tree.MustId())
	require.Nil(t, err)

	objDid, err := inv.DidForName("transfer-test")
	require.Nil(t, err)
	require.NotEmpty(t, objDid)

	stream.ExpectMessage("transfer-test", 5*time.Second)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "look in bag"})
	stream.Wait()

	stream.ExpectMessage("transfer-test has been sent for transfer", 5*time.Second)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "transfer object transfer-test to " + otherPlayerTree.MustId()})
	stream.Wait()

	err = stream.ClearMessages()
	require.Nil(t, err)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "look in bag"})
	time.Sleep(300 * time.Millisecond)
	msgs := filterUserMessages(t, stream.GetMessages())

	includesTransferTestObj := false
	for _, msg := range msgs {
		if strings.Contains(msg.Message, "transfer-test") {
			includesTransferTestObj = true
		}
	}
	require.False(t, includesTransferTestObj)

	stream.ExpectMessage(fmt.Sprintf("transfer-test (%s) has been transferred", objDid), 5*time.Second)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "receive object " + objDid})
	stream.Wait()

	stream.ExpectMessage("transfer-test", 5*time.Second)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "look in bag"})
	stream.Wait()

	err = stream.ClearMessages()
	require.Nil(t, err)
	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "help location"})
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
