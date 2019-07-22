// +build integration

package importer

import (
	"fmt"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	gamepkg "github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/stretchr/testify/require"
)

var rootCtx = actor.EmptyRootContext

func TestImportIntegration(t *testing.T) {
	net := network.NewLocalNetwork()
	stream := ui.NewTestStream()

	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), t.Name()+"-ui")
	require.Nil(t, err)
	playerTree, err := gamepkg.GetOrCreatePlayerTree(net)
	require.Nil(t, err)
	game, err := rootCtx.SpawnNamed(gamepkg.NewGameProps(playerTree, simulatedUI, net), t.Name()+"-game")
	require.Nil(t, err)

	defer rootCtx.Stop(simulatedUI)
	defer rootCtx.Stop(game)

	path := "import-example"
	ids, err := New(net).Import(path)

	rootCtx.Send(game, &jasonsgame.UserInput{Message: fmt.Sprintf("connect location %s as enter importer", ids.Locations["home"])})
	time.Sleep(200 * time.Millisecond)

	cmdsAndResponses := map[string]string{
		"enter importer":                     "you have entered the world of the fairies, in front of you sits a great forest",
		"look around":                        "idol",
		"touch the idol":                     "you probably shouldn't do that",
		"whisper to the idol":                "nomel trebrehs",
		"whisper to the idol sherbert lemon": "object has been picked up",
		"look in bag":                        "idol",
		"enter the forest":                   "you are now in the forest, what now?",
		"take a nap":                         "this is a getvalue interaction",
		"go back":                            "you have entered the world of the fairies, in front of you sits a great forest",
	}

	for cmd, response := range cmdsAndResponses {
		rootCtx.Send(game, &jasonsgame.UserInput{Message: cmd})
		time.Sleep(100 * time.Millisecond)
		require.Contains(t, lastUserMsg(t, stream.GetMessages()), response)
	}
}

func lastUserMsg(t *testing.T, msgs []*jasonsgame.UserInterfaceMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if um := msgs[i].GetUserMessage(); um != nil {
			return um.GetMessage()
		}
	}

	return ""
}
