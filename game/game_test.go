package game

import (
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/davecgh/go-spew/spew"
	"github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavigation(t *testing.T) {
	rootCtx := actor.EmptyRootContext
	simulatedUI, err := rootCtx.SpawnNamed(ui.NewSimulatedUIProps(), "test-navigation-ui")
	require.Nil(t, err)
	defer simulatedUI.Stop()

	net := network.NewLocalNetwork()

	game, err := rootCtx.SpawnNamed(NewGameProps(simulatedUI, net), "test-navigation-game")
	require.Nil(t, err)
	defer game.Stop()

	rootCtx.Send(game, &ui.UserInput{Message: "north"})
	time.Sleep(100 * time.Millisecond)
	fut := rootCtx.RequestFuture(simulatedUI, &ui.GetEventsFromSimulator{}, 1*time.Second)
	evts, err := fut.Result()

	require.Len(t, evts.([]interface{}), 3)
	assert.IsType(t, &navigator.Location{}, evts.([]interface{})[1])
	assert.IsTypef(t, &navigator.Location{}, evts.([]interface{})[2], "evts %s", spew.Sdump(evts))

	// works going back to south
	rootCtx.Send(game, &ui.UserInput{Message: "south"})
	time.Sleep(100 * time.Millisecond)
	fut = rootCtx.RequestFuture(simulatedUI, &ui.GetEventsFromSimulator{}, 1*time.Second)
	evts, err = fut.Result()

	require.Len(t, evts.([]interface{}), 4)
	assert.IsType(t, &navigator.Location{}, evts.([]interface{})[3])

}

func TestSetDescription(t *testing.T) {
	rootCtx := actor.EmptyRootContext
	simulatedUI, err := rootCtx.SpawnNamed(ui.NewSimulatedUIProps(), "test-navigation-ui")
	require.Nil(t, err)
	defer simulatedUI.Stop()

	net := network.NewLocalNetwork()

	game, err := rootCtx.SpawnNamed(NewGameProps(simulatedUI, net), "test-navigation-game")
	require.Nil(t, err)
	defer game.Stop()

	newDescription := "multi word"

	rootCtx.Send(game, &ui.UserInput{Message: "set description " + newDescription})
	time.Sleep(100 * time.Millisecond)

	tree, err := net.GetChainTreeByName("home")
	require.Nil(t, err)
	c := new(navigator.Cursor).SetLocation(0, 0).SetChainTree(tree)
	loc, err := c.GetLocation()
	require.Nil(t, err)
	require.Equal(t, newDescription, loc.Description)
}

func TestSanity(t *testing.T) {
	sw := &safewrap.SafeWrap{}
	n := sw.WrapObject(map[string]string{"tst": "tst"})
	str := n.Cid().String()
	decoded, err := cid.Parse(str)
	require.Nil(t, err)
	assert.Equal(t, n.Cid(), decoded)
}
