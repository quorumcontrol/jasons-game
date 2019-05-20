package game

import (
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavigation(t *testing.T) {
	rootCtx := actor.EmptyRootContext
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), "test-navigation-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(simulatedUI)

	game, err := rootCtx.SpawnNamed(NewGameProps(simulatedUI, net), "test-navigation-game")
	require.Nil(t, err)
	defer rootCtx.Stop(game)

	rootCtx.Send(game, &jasonsgame.UserInput{Message: "north"})
	time.Sleep(100 * time.Millisecond)
	msgs := stream.GetMessages()

	require.Len(t, msgs, 3)

	// works going back to south
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "south"})
	time.Sleep(100 * time.Millisecond)
	msgs = stream.GetMessages()

	require.Len(t, msgs, 4)
	assert.NotNil(t, msgs[3].GetLocation())
}

func TestSetDescription(t *testing.T) {
	rootCtx := actor.EmptyRootContext
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), "test-set-description-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(simulatedUI)

	game, err := rootCtx.SpawnNamed(NewGameProps(simulatedUI, net), "test-set-description-game")
	require.Nil(t, err)
	defer rootCtx.Stop(game)

	newDescription := "multi word"

	rootCtx.Send(game, &jasonsgame.UserInput{Message: "set description " + newDescription})
	time.Sleep(100 * time.Millisecond)

	tree, err := net.GetChainTreeByName("home")
	require.Nil(t, err)
	c := new(navigator.Cursor).SetLocation(0, 0).SetChainTree(tree)
	loc, err := c.GetLocation()
	require.Nil(t, err)
	require.Equal(t, newDescription, loc.Description)
}
func TestBuildPortal(t *testing.T) {
	rootCtx := actor.EmptyRootContext
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), "test-set-description-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(simulatedUI)

	game, err := rootCtx.SpawnNamed(NewGameProps(simulatedUI, net), "test-set-description-game")
	require.Nil(t, err)
	defer rootCtx.Stop(game)

	did := "did:fakedidtonowhere"

	rootCtx.Send(game, &jasonsgame.UserInput{Message: "build portal to " + did})
	time.Sleep(100 * time.Millisecond)

	tree, err := net.GetChainTreeByName("home")
	require.Nil(t, err)
	c := new(navigator.Cursor).SetLocation(0, 0).SetChainTree(tree)
	loc, err := c.GetLocation()
	require.Nil(t, err)
	require.Equal(t, did, loc.Portal.To)
}

func TestGoThroughPortal(t *testing.T) {
	rootCtx := actor.EmptyRootContext
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), "test-set-description-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(simulatedUI)

	game, err := rootCtx.SpawnNamed(NewGameProps(simulatedUI, net), "test-set-description-game")
	require.Nil(t, err)
	defer rootCtx.Stop(game)

	remoteTree, err := net.CreateNamedChainTree("remotetree")
	require.Nil(t, err)
	remoteDescription := "a remote foreign land"
	remoteTree, err = net.UpdateChainTree(remoteTree, "jasons-game/0/0", &jasonsgame.Location{Description: remoteDescription})
	require.Nil(t, err)

	did := remoteTree.MustId()

	rootCtx.Send(game, &jasonsgame.UserInput{Message: "build portal to " + did})
	time.Sleep(100 * time.Millisecond)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: "go through portal"})
	time.Sleep(100 * time.Millisecond)

	msgs := stream.GetMessages()
	lastMsg := msgs[len(msgs)-1]
	assert.Equal(t, remoteDescription, lastMsg.Message)
}
