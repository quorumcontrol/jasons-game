// +build integration

package importer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/config"
	gamepkg "github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/stretchr/testify/require"
)

var rootCtx = actor.EmptyRootContext

func TestImportIntegration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := network.SetupTupeloNotaryGroup(ctx, true)
	require.Nil(t, err)
	net, err := network.NewRemoteNetwork(ctx, group, config.MemoryDataStore())
	require.Nil(t, err)

	stream := ui.NewTestStream(t)
	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), t.Name()+"-ui")
	require.Nil(t, err)
	playerChain, err := net.CreateNamedChainTree("player")
	require.Nil(t, err)
	playerTree, err := gamepkg.CreatePlayerTree(net, playerChain.MustId())
	require.Nil(t, err)

	gameCfg := &gamepkg.GameConfig{PlayerTree: playerTree, UiActor: simulatedUI, Network: net}
	game, err := rootCtx.SpawnNamed(gamepkg.NewGameProps(gameCfg), t.Name()+"-game")
	require.Nil(t, err)

	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	path := "import-example"
	ids, err := New(net).Import(path)
	require.Nil(t, err)
	rootCtx.Send(game, &jasonsgame.UserInput{Message: fmt.Sprintf("connect location %s as enter importer-world", ids.Locations["home"])})
	time.Sleep(10 * time.Millisecond)

	cmdsAndResponses := []string{
		"enter importer-world", "you have entered the world of the fairies, in front of you sits a great forest",
		"look around", "idol",
		"touch the idol", "you probably shouldn't do that",
		"whisper to the idol", "nomel trebrehs",
		"whisper to the idol sherbert lemon", "object has been picked up",
		"look in bag", "idol",
		"enter the forest", "you are now in the forest, what now?",
		"take a nap", "this is a getvalue interaction",
		"go back", "you have entered the world of the fairies, in front of you sits a great forest",
	}

	for i := 0; i < len(cmdsAndResponses); i = i + 2 {
		cmd := cmdsAndResponses[i]
		response := cmdsAndResponses[i+1]

		stream.ExpectMessage(response, 3*time.Second)
		rootCtx.Send(game, &jasonsgame.UserInput{Message: cmd})
		stream.Wait()
	}
}
