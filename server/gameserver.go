package server

import (
	"context"
	"log"
	"os"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
)

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
	os.MkdirAll(statePath, 0755)
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
	log.Printf("received: %v", inp)
	act := gs.getOrCreateSession(inp.Session, nil)

	actor.EmptyRootContext.Send(act, inp)
	return &jasonsgame.CommandReceived{}, nil
}

func (gs *GameServer) ReceiveUserMessages(sess *jasonsgame.Session, stream jasonsgame.GameService_ReceiveUserMessagesServer) error {
	log.Printf("receive user messages %v", sess)

	// TODO: do we want to do anything here?
	gs.getOrCreateSession(sess, stream)
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
			panic("must supply a valid session")
		}
		log.Printf("creating actor")
		uiActor = actor.EmptyRootContext.Spawn(ui.NewUIProps(stream, gs.network))

		actor.EmptyRootContext.Spawn(game.NewGameProps(uiActor, gs.network))

		gs.sessions[sess.Uuid] = uiActor
	}
	return uiActor
}
