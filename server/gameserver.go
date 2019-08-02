package server

import (
	"context"
	"os"
	"path/filepath"
	"time"

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

func (gs *GameServer) getOrCreateSession(sess *jasonsgame.Session, stream jasonsgame.GameService_ReceiveUIMessagesServer) *actor.PID {
	uiActor, ok := gs.sessions[sess.Uuid]
	if !ok {
		// use filepath.Base as a "cleaner" here to not allow setting arbitrary directories with, for example, uuid: "../../etc/passwd"
		statePath := filepath.Join(gs.sessionPath, filepath.Base(sess.Uuid))
		if err := os.MkdirAll(statePath, 0750); err != nil {
			panic(errors.Wrap(err, "error creating session storage"))
		}

		ds, err := config.LocalDataStore(statePath)
		if err != nil {
			panic(errors.Wrap(err, "error getting store"))
		}

		net, err := network.NewRemoteNetwork(gs.parentCtx, gs.group, ds)
		if err != nil {
			panic(errors.Wrap(err, "setting up network"))
		}

		if sess == nil {
			// TODO: do this more gracefully
			log.Errorf("no session")
			panic("must supply a valid session")
		}

		log.Debugf("creating actors")
		uiActor = actor.EmptyRootContext.Spawn(ui.NewUIProps(stream, net))
		gs.sessions[sess.Uuid] = uiActor

		playerTree, err := game.GetPlayerTree(net)
		if err != nil {
			panic(errors.Wrap(err, "error getting player tree"))
		}

		gameCfg := &game.GameConfig{
			PlayerTree: playerTree,
			UiActor:    uiActor,
			Network:    net,
			InkDID:     gs.inkDID,
		}
		actor.EmptyRootContext.Spawn(game.NewGameProps(gameCfg))
	}
	return uiActor
}
