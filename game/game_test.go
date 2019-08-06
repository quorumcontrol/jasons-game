package game

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/game/trees"
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

func filterUserMessages(t *testing.T, msgs []*jasonsgame.UserInterfaceMessage) []*jasonsgame.MessageToUser {
	usrMsgs := make([]*jasonsgame.MessageToUser, 0)
	for _, m := range msgs {
		if um := m.GetUserMessage(); um != nil {
			usrMsgs = append(usrMsgs, um)
		}
	}

	return usrMsgs
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

	usrMsgs := filterUserMessages(t, msgs)
	require.Len(t, usrMsgs, 3)

	// Local network doesn't auto-refresh commands because that is tied to a tupelo
	// state refresh
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "refresh"})
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "enter dungeon"})
	time.Sleep(100 * time.Millisecond)
	msgs = stream.GetMessages()

	usrMsgs = filterUserMessages(t, msgs)

	require.Len(t, usrMsgs, 5)
	assert.NotNil(t, usrMsgs[3].GetLocation())
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
		if userMessage := msg.GetUserMessage(); userMessage != nil {
			if strings.Contains(msg.GetUserMessage().Message, newDescription) {
				respondedWithDescription = true
			}
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
		if strings.Contains(msg.GetUserMessage().Message, did) {
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
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "build portal to" + did})
	time.Sleep(100 * time.Millisecond)
	// Local network doesn't auto-refresh commands because that is tied to a tupelo
	// state refresh
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "refresh"})

	err = stream.ClearMessages()
	require.Nil(t, err)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "go through portal"})
	time.Sleep(100 * time.Millisecond)

	msgs := stream.GetMessages()
	lastMsg := msgs[len(msgs)-1]
	assert.Equal(t, remoteDescription, lastMsg.GetUserMessage().Message)
}

func TestInscriptionInteractions(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, game := setupUiAndGame(t, stream, net)
	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	playerTree, err := GetOrCreatePlayerTree(net)
	require.Nil(t, err)

	objTree, err := net.CreateChainTree()
	require.Nil(t, err)
	obj := NewObjectTree(net, objTree)
	err = obj.SetName("test-object")
	require.Nil(t, err)

	inventoryTree, err := trees.FindInventoryTree(net, playerTree.Did())
	require.Nil(t, err)
	err = inventoryTree.Add(obj.MustId())
	require.Nil(t, err)

	t.Run("with a mutli-valued inscription", func(t *testing.T) {
		err = obj.AddInteraction(&SetTreeValueInteraction{
			Command:  "test1 inscribe",
			Did:      obj.MustId(),
			Path:     "inscriptions",
			Multiple: true,
		})
		require.Nil(t, err)

		err = obj.AddInteraction(&GetTreeValueInteraction{
			Command: "test1 read",
			Did:     obj.MustId(),
			Path:    "inscriptions",
		})
		require.Nil(t, err)
		time.Sleep(50 * time.Millisecond)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "refresh"})
		time.Sleep(50 * time.Millisecond)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test1 inscribe this is a magic sword"})
		time.Sleep(50 * time.Millisecond)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test1 inscribe with magical properties"})
		time.Sleep(50 * time.Millisecond)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test1 read inscriptions"})
		time.Sleep(50 * time.Millisecond)
		msgs := filterUserMessages(t, stream.GetMessages())
		lastMsg := msgs[len(msgs)-1]
		assert.Equal(t, lastMsg.Message, "this is a magic sword\nwith magical properties")
	})

	t.Run("with a mutli-valued inscription", func(t *testing.T) {
		err = obj.AddInteraction(&SetTreeValueInteraction{
			Command: "test2 inscribe",
			Did:     obj.MustId(),
			Path:    "inscriptions2",
		})
		require.Nil(t, err)

		err = obj.AddInteraction(&GetTreeValueInteraction{
			Command: "test2 read",
			Did:     obj.MustId(),
			Path:    "inscriptions2",
		})
		require.Nil(t, err)
		time.Sleep(50 * time.Millisecond)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "refresh"})
		time.Sleep(50 * time.Millisecond)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test2 inscribe this is a magic sword"})
		time.Sleep(50 * time.Millisecond)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test2 inscribe with magical properties"})
		time.Sleep(50 * time.Millisecond)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test2 read inscriptions"})
		time.Sleep(50 * time.Millisecond)
		msgs := filterUserMessages(t, stream.GetMessages())
		lastMsg := msgs[len(msgs)-1]
		assert.Equal(t, lastMsg.Message, "with magical properties")
	})
}

func TestCantDropFromOtherTree(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, game := setupUiAndGame(t, stream, net)
	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	playerTree, err := GetPlayerTree(net)
	require.Nil(t, err)

	objTree, err := net.CreateChainTree()
	require.Nil(t, err)

	obj := NewObjectTree(net, objTree)
	err = obj.SetName("obj-to-drop")
	require.Nil(t, err)

	err = obj.AddInteraction(&DropObjectInteraction{
		Command: "drop will succeed",
		Did:     obj.MustId(),
	})
	require.Nil(t, err)

	otherObjTree, err := net.CreateChainTree()
	require.Nil(t, err)
	otherObj := NewObjectTree(net, otherObjTree)
	err = otherObj.SetName("obj-triggering-drop")
	require.Nil(t, err)

	err = otherObj.AddInteraction(&DropObjectInteraction{
		Command: "drop will fail",
		Did:     obj.MustId(),
	})
	require.Nil(t, err)

	inventoryTree, err := trees.FindInventoryTree(net, playerTree.Did())
	require.Nil(t, err)
	err = inventoryTree.Add(obj.MustId())
	require.Nil(t, err)
	err = inventoryTree.Add(otherObj.MustId())
	require.Nil(t, err)

	time.Sleep(50 * time.Millisecond)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "drop will fail"})
	time.Sleep(50 * time.Millisecond)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "look around"})
	time.Sleep(50 * time.Millisecond)

	msgs := filterUserMessages(t, stream.GetMessages())
	assert.NotContains(t, msgs[len(msgs)-1].Message, "obj-to-drop")

	time.Sleep(50 * time.Millisecond)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "drop will succeed"})
	time.Sleep(50 * time.Millisecond)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "look around"})
	time.Sleep(50 * time.Millisecond)
	msgs = filterUserMessages(t, stream.GetMessages())
	assert.Contains(t, msgs[len(msgs)-1].Message, "obj-to-drop")
}
