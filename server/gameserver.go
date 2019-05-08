package server

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
)

var log = logging.Logger("gameserver")

const statePath = "/tmp/jasonsgame"

type GameServer struct {
	sessions map[string]*actor.PID
	network  network.Network
}

func NewGameServer(ctx context.Context) *GameServer {
	group, err := setupNotaryGroup(ctx)
	if err != nil {
		panic(errors.Wrap(err, "setting up notary group"))
	}

	os.RemoveAll(statePath)
	err = os.MkdirAll(statePath, 0755)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error creating state path %s", statePath)))
	}
	defer os.RemoveAll(statePath)

	net, err := network.NewRemoteNetwork(ctx, group, statePath)
	if err != nil {
		panic(errors.Wrap(err, "setting up notary group"))
	}

	return &GameServer{
		sessions: make(map[string]*actor.PID),
		network:  net,
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

func (gs *GameServer) ReceiveUserMessages(sess *jasonsgame.Session, stream jasonsgame.GameService_ReceiveUserMessagesServer) error {
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

func (gs *GameServer) getOrCreateSession(sess *jasonsgame.Session, stream jasonsgame.GameService_ReceiveUserMessagesServer) *actor.PID {
	uiActor, ok := gs.sessions[sess.Uuid]
	if !ok {
		if sess == nil {
			// TODO: do this more gracefully
			log.Errorf("no session")
			panic("must supply a valid session")
		}
		log.Debugf("creating actor")
		uiActor = actor.EmptyRootContext.Spawn(ui.NewUIProps(stream, gs.network))

		actor.EmptyRootContext.Spawn(game.NewGameProps(uiActor, gs.network))

		gs.sessions[sess.Uuid] = uiActor
	}
	return uiActor
}
