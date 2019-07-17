package ui

import (
	"fmt"
	"reflect"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/gogo/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

var log = logging.Logger("uiserver")

type remoteStream interface {
	Send(*jasonsgame.UserInterfaceMessage) error
}

type TestStream struct {
	messages []*jasonsgame.UserInterfaceMessage
}

func NewTestStream() *TestStream {
	return &TestStream{}
}

func (ts *TestStream) Send(msg *jasonsgame.UserInterfaceMessage) error {
	ts.messages = append(ts.messages, msg)
	return nil
}

func (ts *TestStream) GetMessages() []*jasonsgame.UserInterfaceMessage {
	return ts.messages
}

func (ts *TestStream) ClearMessages() error {
	ts.messages = NewTestStream().messages
	return nil
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

func buildUIMessage(msg proto.Message) (*jasonsgame.UserInterfaceMessage, error) {
	switch msg := msg.(type) {
	case *jasonsgame.MessageToUser:
		uiMsg := &jasonsgame.UserInterfaceMessage{
			UiMessage: &jasonsgame.UserInterfaceMessage_UserMessage{UserMessage: msg},
		}

		return uiMsg, nil
	case *jasonsgame.CommandUpdate:
		uiMsg := &jasonsgame.UserInterfaceMessage{
			UiMessage: &jasonsgame.UserInterfaceMessage_CommandUpdate{CommandUpdate: msg},
		}
		return uiMsg, nil
	case *jasonsgame.ImportResult:
		uiMsg := &jasonsgame.UserInterfaceMessage{
			UiMessage: &jasonsgame.UserInterfaceMessage_ImportResult{ImportResult: msg},
		}
		return uiMsg, nil

	default:
		return nil, fmt.Errorf("Unrecognized user interface message: %v", msg)
	}
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

		if us.game != nil {
			cmdUpdate := &jasonsgame.CommandUpdate{}
			fut := actorCtx.RequestFuture(us.game, cmdUpdate, 5*time.Second)
			res, err := fut.Result()
			if err != nil {
				log.Errorf("error waiting for future: %v", err)
			}
			log.Debugf("received response from game: %v", res)
		}

	case *jasonsgame.MessageToUser:
		actorCtx.SetReceiveTimeout(5 * time.Second)
		log.Debugf("message to user: %s", msg.Message)
		if us.stream == nil {
			log.Errorf("no valid stream for user message: %v", msg.Message)
			return
		}

		uiMsg, err := buildUIMessage(msg)
		if err != nil {
			panic(err)
		}

		err = us.stream.Send(uiMsg)
		if err != nil {
			us.stream = nil
			us.sendDone()
			log.Errorf("error sending message to stream: %v", err)
		}

	case *jasonsgame.ImportResult:
		actorCtx.SetReceiveTimeout(5 * time.Second)
		log.Debugf("import result: %s", msg)
		if us.stream == nil {
			log.Errorf("no valid stream for import result: %v", msg)
			return
		}

		uiMsg, err := buildUIMessage(msg)
		if err != nil {
			panic(err)
		}

		err = us.stream.Send(uiMsg)
		if err != nil {
			us.stream = nil
			us.sendDone()
			log.Errorf("error sending import result to stream: %v", err)
		}

	case *jasonsgame.CommandUpdate:
		actorCtx.SetReceiveTimeout(5 * time.Second)
		log.Debugf("command update: %s", msg.Commands)
		if us.stream == nil {
			log.Errorf("no valid stream for command update: %v", msg.Commands)
			return
		}

		uiMsg, err := buildUIMessage(msg)
		if err != nil {
			panic(err)
		}

		err = us.stream.Send(uiMsg)
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
	case *jasonsgame.ImportRequest:
		actorCtx.SetReceiveTimeout(5 * time.Second)
		log.Debugf("import request", msg)
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
		log.Debugf("import has no game to go to %v", msg)
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
