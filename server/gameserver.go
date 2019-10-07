package server

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/99designs/keyring"
	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/game"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
)

var log = logging.Logger("gameserver")

const sessionStorageDir = "session-storage"

type GameServer struct {
	sessions    map[string]*actor.PID
	group       *types.NotaryGroup
	parentCtx   context.Context
	sessionPath string
	inkDID      string
}

type GameServerConfig struct {
	LocalNet bool
	InkDID   string
}

func NewGameServer(ctx context.Context, cfg GameServerConfig) *GameServer {
	group, err := network.SetupTupeloNotaryGroup(ctx, cfg.LocalNet)
	if err != nil {
		panic(errors.Wrap(err, "setting up notary group"))
	}

	sessionCfg := config.EnsureExists(sessionStorageDir)

	return &GameServer{
		sessions:    make(map[string]*actor.PID),
		group:       group,
		parentCtx:   ctx,
		sessionPath: sessionCfg.Path,
		inkDID:      cfg.InkDID,
	}
}

func (gs *GameServer) SendCommand(ctx context.Context, inp *jasonsgame.UserInput) (*jasonsgame.CommandReceived, error) {
	log.Debugf("received: %v", inp)
	act := gs.getOrCreateSession(inp.Session, nil)
	if act == nil {
		log.Errorf("error, received nil actor for session %v", inp.Session)
	}

	fut := actor.EmptyRootContext.RequestFuture(act, inp, 5*time.Second)
	res, err := fut.Result()
	if err != nil {
		log.Errorf("error waiting for UI: %v", err)
		return nil, errors.Wrap(err, "error waiting on command input")
	}
	return res.(*jasonsgame.CommandReceived), nil
}

func (gs *GameServer) ReceiveUIMessages(sess *jasonsgame.Session, stream jasonsgame.GameService_ReceiveUIMessagesServer) error {
	log.Debugf("receive user messages %v", sess)

	act := gs.getOrCreateSession(sess, stream)

	ch := make(chan struct{})
	actor.EmptyRootContext.Send(act, &ui.SetStream{Stream: stream, DoneChan: ch})
	<-ch
	// we need to keep this request open until we are done
	return nil
}

func (gs *GameServer) ReceiveStatMessages(sess *jasonsgame.Session, stream jasonsgame.GameService_ReceiveStatMessagesServer) error {
	return nil
}

var secureKeyringBackends = []keyring.BackendType{
	keyring.WinCredBackend,
	keyring.KeychainBackend,
	keyring.SecretServiceBackend,
	keyring.KWalletBackend,
	keyring.PassBackend,
}

const insecureKeyringMessage = "WARNING: Jasons' Game was unable to find a secure keystore on your system.\nPlease consider using one of the backends listed here: https://github.com/99designs/keyring"

func (gs *GameServer) getOrCreateSession(sess *jasonsgame.Session, stream jasonsgame.GameService_ReceiveUIMessagesServer) *actor.PID {
	uiActor, ok := gs.sessions[sess.Uuid]

	if !ok {
		if sess == nil {
			// TODO: do this more gracefully
			log.Errorf("no session")
			panic("must supply a valid session")
		}

		// use filepath.Base as a "cleaner" here to not allow setting arbitrary directories with, for example, uuid: "../../etc/passwd"
		statePath := filepath.Join(gs.sessionPath, filepath.Base(sess.Uuid))
		if err := os.MkdirAll(statePath, 0750); err != nil {
			panic(errors.Wrap(err, "error creating session storage"))
		}

		ds, err := config.LocalDataStore(filepath.Join(statePath, "data"))
		if err != nil {
			panic(errors.Wrap(err, "error getting store"))
		}

		log.Debugf("creating actors")
		uiActor = actor.EmptyRootContext.Spawn(ui.NewUIProps(stream))
		gs.sessions[sess.Uuid] = uiActor

		kr, err := keyring.Open(keyring.Config{
			ServiceName:                    "Jasons Game",
			KeychainTrustApplication:       true,
			KeychainAccessibleWhenUnlocked: true,
			AllowedBackends:                secureKeyringBackends,
		})

		// Fallback to insecure file store, warn user
		if kr == nil {
			actor.EmptyRootContext.Send(uiActor, &jasonsgame.MessageToUser{Message: insecureKeyringMessage})
			kr, err = keyring.Open(keyring.Config{
				FileDir:         filepath.Join(statePath, "keys"),
				AllowedBackends: []keyring.BackendType{keyring.FileBackend},
				FilePasswordFunc: func(_ string) (string, error) {
					return "insecure", nil
				},
			})
		}

		if err != nil {
			panic(errors.Wrap(err, "error opening keyring"))
		}

		actor.EmptyRootContext.Spawn(game.NewAuthenticatedSessionProps(gs.parentCtx, &game.AuthenticatedSessionConfig{
			UiActor:     uiActor,
			DataStore:   ds,
			NotaryGroup: gs.group,
			Keyring:     kr,
		}))
	}
	return uiActor
}
