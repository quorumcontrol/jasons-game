package ui

import (
	"log"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

type remoteStream interface {
	Send(*jasonsgame.MessageToUser) error
}

type TestStream struct {
	messages []*jasonsgame.MessageToUser
}

func NewTestStream() *TestStream {
	return &TestStream{}
}

func (ts *TestStream) Send(msg *jasonsgame.MessageToUser) error {
	ts.messages = append(ts.messages, msg)
	return nil
}

func (ts *TestStream) GetMessages() []*jasonsgame.MessageToUser {
	return ts.messages
}

type UIServer struct {
	game    *actor.PID
	network network.Network
	stream  remoteStream
}

func NewUIProps(stream remoteStream, net network.Network) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &UIServer{
			network: net,
			stream:  stream,
		}
	})
}

type SetGame struct {
	Game *actor.PID
}

func (us *UIServer) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Stopping:
		if us.game != nil {
			us.game.Poison()
		}
	case *SetGame:
		us.game = msg.Game
	case *jasonsgame.MessageToUser:
		log.Printf("message to user: %s", msg.Message)
		us.stream.Send(msg)
	case *jasonsgame.UserInput:
		log.Printf("user input %s", msg.Message)
		if us.game != nil {
			actorCtx.Send(us.game, msg)
			return
		}
		log.Printf("user input has no game to go to %v", msg)
	}
}
