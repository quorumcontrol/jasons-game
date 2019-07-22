// +build integration

package importer

import (
	"fmt"
	"strings"
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

		rootCtx.Send(game, &jasonsgame.UserInput{Message: cmd})

		found := false
		for !found {
			select {
			case msg := <-stream.Channel():
				if um := msg.GetUserMessage(); um != nil {
					found = strings.Contains(um.GetMessage(), response)
				}
			case <-time.After(1 * time.Second):
				require.Fail(t, fmt.Sprintf("Timeout waiting for command: %s\nexpected response: %s\nmessages: %v", cmd, response, stream.GetMessages()))
			}
		}
	}
}
