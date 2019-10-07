// +build integration

package game

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/99designs/keyring"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
)

func TestAuthenticatedSession(t *testing.T) {
	stopFn := func(cancelFn context.CancelFunc, session *actor.PID) {
		remote.Stop()
		time.Sleep(100 * time.Millisecond)
		if cancelFn != nil {
			cancelFn()
		}
		time.Sleep(100 * time.Millisecond)
		if session != nil {
			_ = rootCtx.StopFuture(session).Wait()
		}
		time.Sleep(1 * time.Second)
	}

	group, err := network.SetupTupeloNotaryGroup(context.Background(), true)
	require.Nil(t, err)

	stream := ui.NewTestStream(t)
	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream), t.Name()+"-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(simulatedUI)

	dir, err := ioutil.TempDir(os.TempDir(), "jg-test")
	require.Nil(t, err)
	defer os.RemoveAll(dir)

	testKeyring, err := keyring.Open(keyring.Config{
		FileDir:         dir,
		AllowedBackends: []keyring.BackendType{keyring.FileBackend},
		FilePasswordFunc: func(_ string) (string, error) {
			return "integration-test", nil
		},
	})
	require.Nil(t, err)

	ds := config.MemoryDataStore()

	ctx1, cancel1 := context.WithCancel(context.Background())
	stream.ExpectMessage(loginWelcomeMessage, 5*time.Second)
	session1 := rootCtx.Spawn(NewAuthenticatedSessionProps(ctx1, &AuthenticatedSessionConfig{
		UiActor:     simulatedUI,
		DataStore:   ds,
		NotaryGroup: group,
		Keyring:     testKeyring,
	}))
	defer stopFn(cancel1, session1)
	stream.Wait()

	stream.ExpectMessage(fmt.Sprintf(loginEmailErrorMessage, "sign up"), 5*time.Second)
	rootCtx.Send(session1, &jasonsgame.UserInput{Message: "sign up notavalidemail"})
	stream.Wait()

	stream.ExpectMessage("Signing up with test@localhost", 5*time.Second)
	rootCtx.Send(session1, &jasonsgame.UserInput{Message: "sign up test@localhost"})
	stream.Wait()

	msgs := filterUserMessages(t, stream.GetMessages())
	recoveryMessage := msgs[len(msgs)-1]
	recoveryPhrase := strings.Split(strings.Join(strings.Split(recoveryMessage.Message, "\n")[4:6], " "), " ")
	assert.Len(t, recoveryPhrase, 24)

	// Since there is no static tree or arcadia, the game will fallback to player home
	stream.ExpectMessage(homeLocationDescription, 5*time.Second)
	rootCtx.Send(session1, &jasonsgame.UserInput{Message: "portal to fae"})
	stream.Wait()

	stopFn(cancel1, session1)

	// start a new session, should use existing key and launch straight to game
	ctx2, cancel2 := context.WithCancel(context.Background())
	stream.ExpectMessage(homeLocationDescription, 5*time.Second)
	session2 := rootCtx.Spawn(NewAuthenticatedSessionProps(ctx2, &AuthenticatedSessionConfig{
		UiActor:     simulatedUI,
		DataStore:   ds,
		NotaryGroup: group,
		Keyring:     testKeyring,
	}))
	stream.Wait()
	stopFn(cancel2, session2)

	// start a new session & remove private key in order to test recovery
	err = testKeyring.Remove(keyringPrivateKeyName)
	require.Nil(t, err)

	ctx3, cancel3 := context.WithCancel(context.Background())
	stream.ExpectMessage(loginWelcomeMessage, 5*time.Second)
	session3 := rootCtx.Spawn(NewAuthenticatedSessionProps(ctx3, &AuthenticatedSessionConfig{
		UiActor:     simulatedUI,
		DataStore:   config.MemoryDataStore(),
		NotaryGroup: group,
		Keyring:     testKeyring,
	}))
	defer stopFn(cancel3, session3)
	stream.Wait()

	// recover with wrong email should fail
	stream.ExpectMessage(loginRecoveryMessage, 5*time.Second)
	rootCtx.Send(session3, &jasonsgame.UserInput{Message: "recover test-bad@localhost"})
	stream.Wait()

	stream.ExpectMessage(loginRecoveryFailureMessage, 5*time.Second)
	rootCtx.Send(session3, &jasonsgame.UserInput{Message: fmt.Sprintf("recovery phrase %s", strings.Join(recoveryPhrase, " "))})
	stream.Wait()

	// recover with wrong seed phrase should fail
	stream.ExpectMessage(loginRecoveryMessage, 5*time.Second)
	rootCtx.Send(session3, &jasonsgame.UserInput{Message: "recover test@localhost"})
	stream.Wait()

	stream.ExpectMessage(loginRecoveryFailureMessage, 5*time.Second)
	rootCtx.Send(session3, &jasonsgame.UserInput{Message: fmt.Sprintf("recovery phrase notaseedword %s", strings.Join(recoveryPhrase[0:22], " "))})
	stream.Wait()

	// entering a correct email and recovery phrase should forward user into game
	stream.ExpectMessage(loginRecoveryMessage, 5*time.Second)
	rootCtx.Send(session3, &jasonsgame.UserInput{Message: "recover test@localhost"})
	stream.Wait()

	stream.ExpectMessage(homeLocationDescription, 5*time.Second)
	rootCtx.Send(session3, &jasonsgame.UserInput{Message: fmt.Sprintf("recovery phrase %s", strings.Join(recoveryPhrase, " "))})
	stream.Wait()
}
