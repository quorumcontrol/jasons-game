package ui

import (
	"reflect"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

var log = logging.Logger("uiserver")

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

type doneChan chan struct{}

type UIServer struct {
	game     *actor.PID
	network  network.Network
	stream   remoteStream
	doneChan doneChan
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

type SetStream struct {
	Stream   remoteStream
	DoneChan doneChan
}

func (us *UIServer) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Stopping:
		if us.game != nil {
			actorCtx.Poison(us.game)
		}
		if us.doneChan != nil {
			us.doneChan <- struct{}{}
		}
	case *actor.ReceiveTimeout:
		actorCtx.Send(actorCtx.Self(), &jasonsgame.MessageToUser{Heartbeat: true})
	case *SetGame:
		us.game = msg.Game
	case *SetStream:
		// free up the previous stream
		us.sendDone()

		us.stream = msg.Stream
		us.doneChan = msg.DoneChan
		actorCtx.Send(actorCtx.Self(), &jasonsgame.MessageToUser{Message: "missed you while you were gone"})
	case *jasonsgame.MessageToUser:
		actorCtx.SetReceiveTimeout(5 * time.Second)
		log.Debugf("message to user: %s", msg.Message)
		if us.stream == nil {
			log.Errorf("no valid stream for %v", msg.Message)
			return
		}
		err := us.stream.Send(msg)
		if err != nil {
			us.stream = nil
			us.sendDone()
			log.Errorf("error sending message to stream: %v", err)
		}

	case *jasonsgame.UserInput:
		actorCtx.SetReceiveTimeout(5 * time.Second)
		log.Debugf("user input %s", msg.Message)
		if us.game != nil {
			fut := actorCtx.RequestFuture(us.game, msg, 5*time.Second)
			res, err := fut.Result()
			if err != nil {
				log.Errorf("error waiting for future: %v", err)
			}
			log.Debugf("received response from game")
			if sender := actorCtx.Sender(); sender != nil {
				actorCtx.Respond(res)
			}
			return
		}
		log.Debugf("user input has no game to go to %v", msg)
	default:
		log.Debugf("received unknown message: %v (%s)", msg, reflect.TypeOf(msg).String())
	}
}

func (us *UIServer) sendDone() {
	if us.doneChan != nil {
		select {
		case us.doneChan <- struct{}{}:
			log.Debugf("sent done")
		default:
			log.Warningf("nothing listening on done channel")
		}
	}
}