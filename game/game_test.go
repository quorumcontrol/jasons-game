package game

import (
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/stretchr/testify/require"
)

var rootCtx = actor.EmptyRootContext

func setupUiAndGame(t *testing.T, stream *ui.TestStream, net network.Network) (simulatedUI, game *actor.PID) {
	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), t.Name()+"-ui")
	require.Nil(t, err)

	playerChain, err := net.CreateNamedChainTree("player")
	require.Nil(t, err)
	playerTree, err := CreatePlayerTree(net, playerChain.MustId())
	require.Nil(t, err)

	gameCfg := &GameConfig{PlayerTree: playerTree, UiActor: simulatedUI, Network: net}
	game, err = rootCtx.SpawnNamed(NewGameProps(gameCfg), t.Name()+"-game")
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

func TestSetDescription(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream(t)

	simulatedUI, game := setupUiAndGame(t, stream, net)
	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	newDescription := "multi word"

	stream.ExpectMessage(newDescription, 2*time.Second)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "set description " + newDescription})
	stream.Wait()
}

func TestCallMe(t *testing.T) {
	rootCtx := actor.EmptyRootContext
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream(t)

	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), "test-callme-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(simulatedUI)

	playerChain, err := net.CreateNamedChainTree("player")
	require.Nil(t, err)
	playerTree, err := CreatePlayerTree(net, playerChain.MustId())
	require.Nil(t, err)

	gameCfg := &GameConfig{PlayerTree: playerTree, UiActor: simulatedUI, Network: net}
	game, err := rootCtx.SpawnNamed(NewGameProps(gameCfg), "test-callme-game")
	require.Nil(t, err)
	defer rootCtx.Stop(game)

	newName := "Johnny B Good"

	rootCtx.Send(game, &jasonsgame.UserInput{Message: "call me " + newName})
	time.Sleep(100 * time.Millisecond)

	tree, err := net.GetTree(playerChain.MustId())
	require.Nil(t, err)

	pt := NewPlayerTree(net, tree)
	player, err := pt.Player()
	require.Nil(t, err)
	require.Equal(t, newName, player.Name)
}

func TestBuildPortal(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream(t)

	simulatedUI, game := setupUiAndGame(t, stream, net)
	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	did := "did:fakedidtonowhere"

	stream.ExpectMessage("built a portal", 2*time.Second)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "build portal to " + did})
	stream.Wait()

	stream.ExpectMessage(did, 2*time.Second)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "look around"})
	stream.Wait()
}

func TestGoThroughPortal(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream(t)

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
	stream.ExpectMessage("built a portal", 2*time.Second)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "build portal to" + did})
	stream.Wait()

	rootCtx.Send(game, &jasonsgame.UserInput{Message: "refresh"})

	stream.ExpectMessage(remoteDescription, 2*time.Second)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "go through portal"})
	stream.Wait()
}

func TestInscriptionInteractions(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream(t)

	simulatedUI, game := setupUiAndGame(t, stream, net)
	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	playerTree, err := GetPlayerTree(net)
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
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test1 inscribe this is a magic sword"})
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test1 inscribe with magical properties"})

		stream.ExpectMessage("this is a magic sword\nwith magical properties", 2*time.Second)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test1 read inscriptions"})
		stream.Wait()
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
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test2 inscribe this is a magic sword"})
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test2 inscribe with magical properties"})

		stream.ExpectMessage("with magical properties", 2*time.Second)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: "test2 read inscriptions"})
		stream.Wait()
	})
}

func TestCantDropFromOtherTree(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream(t)

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

	rootCtx.Send(game, &jasonsgame.UserInput{Message: "refresh"})

	stream.ExpectMessage("not allowed", 2*time.Second)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "drop will fail"})
	stream.Wait()

	stream.ExpectMessage("obj-to-drop", 2*time.Second)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "drop will succeed"})
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "look around"})
	stream.Wait()
}
