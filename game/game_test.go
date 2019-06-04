package game

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rootCtx = actor.EmptyRootContext

func setupUiAndGame(t *testing.T, stream *ui.TestStream, net network.Network) (simulatedUI, game *actor.PID) {
	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), t.Name()+"-ui")
	require.Nil(t, err)

	playerTree, err := GetOrCreatePlayerTree(net)
	require.Nil(t, err)
	game, err = rootCtx.SpawnNamed(NewGameProps(playerTree, simulatedUI, net), t.Name()+"-game")
	require.Nil(t, err)
	return simulatedUI, game
}

func TestNavigation(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, game := setupUiAndGame(t, stream, net)
	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	someTree, err := net.CreateChainTree()
	require.Nil(t, err)

	rootCtx.Send(game, &jasonsgame.UserInput{Message: fmt.Sprintf("connect location %s as enter dungeon", someTree.MustId())})
	time.Sleep(100 * time.Millisecond)
	msgs := stream.GetMessages()
	require.Len(t, msgs, 3)

	rootCtx.Send(game, &jasonsgame.UserInput{Message: "enter dungeon"})
	time.Sleep(100 * time.Millisecond)
	msgs = stream.GetMessages()

	require.Len(t, msgs, 4)
	assert.NotNil(t, msgs[3].GetLocation())
}

func TestSetDescription(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, game := setupUiAndGame(t, stream, net)
	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	newDescription := "multi word"

	rootCtx.Send(game, &jasonsgame.UserInput{Message: "set description " + newDescription})
	time.Sleep(100 * time.Millisecond)

	respondedWithDescription := false
	for _, msg := range stream.GetMessages() {
		if strings.Contains(msg.Message, newDescription) {
			respondedWithDescription = true
		}
	}
	require.True(t, respondedWithDescription)
}

func TestCallMe(t *testing.T) {
	rootCtx := actor.EmptyRootContext
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), "test-callme-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(simulatedUI)

	playerTree, err := GetOrCreatePlayerTree(net)
	require.Nil(t, err)
	game, err := rootCtx.SpawnNamed(NewGameProps(playerTree, simulatedUI, net), "test-callme-game")
	require.Nil(t, err)
	defer rootCtx.Stop(game)

	newName := "Johnny B Good"

	rootCtx.Send(game, &jasonsgame.UserInput{Message: "call me " + newName})
	time.Sleep(100 * time.Millisecond)

	tree, err := net.GetChainTreeByName("player")
	require.Nil(t, err)

	pt := NewPlayerTree(net, tree)
	player, err := pt.Player()
	require.Nil(t, err)
	require.Equal(t, newName, player.Name)
}

func TestBuildPortal(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, game := setupUiAndGame(t, stream, net)
	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	did := "did:fakedidtonowhere"
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "build portal to " + did})
	time.Sleep(100 * time.Millisecond)

	tree, err := net.GetChainTreeByName("home")
	require.Nil(t, err)
	loc := NewLocationTree(net, tree)
	portal, err := loc.GetPortal()
	require.Nil(t, err)
	require.Equal(t, portal.To, did)

	err = stream.ClearMessages()
	require.Nil(t, err)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "look around"})
	time.Sleep(100 * time.Millisecond)

	respondedWithPortal := false
	for _, msg := range stream.GetMessages() {
		if strings.Contains(msg.Message, did) {
			respondedWithPortal = true
		}
	}
	require.True(t, respondedWithPortal)
}

func TestGoThroughPortal(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, game := setupUiAndGame(t, stream, net)
	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	remoteTree, err := net.CreateChainTree()
	require.Nil(t, err)
	loc := NewLocationTree(net, remoteTree)
	remoteDescription := "a remote foreign land"
	err = loc.SetDescription(remoteDescription)
	require.Nil(t, err)

	did := remoteTree.MustId()
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "build portal to " + did})
	time.Sleep(100 * time.Millisecond)

	err = stream.ClearMessages()
	require.Nil(t, err)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "go through portal"})
	time.Sleep(100 * time.Millisecond)

	msgs := stream.GetMessages()
	lastMsg := msgs[len(msgs)-1]
	assert.Equal(t, remoteDescription, lastMsg.Message)
}
