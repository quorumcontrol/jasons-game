package ui

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/gogo/protobuf/proto"
	logging "github.com/ipfs/go-log"

	"github.com/quorumcontrol/jasons-game/inkfaucet/ink"
	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
	"github.com/quorumcontrol/jasons-game/inkfaucet/invites"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

var log = logging.Logger("uiserver")

type remoteStream interface {
	Send(*jasonsgame.UserInterfaceMessage) error
}

type TestStream struct {
	messages []*jasonsgame.UserInterfaceMessage
	channel  chan *jasonsgame.UserInterfaceMessage
}

func NewTestStream() *TestStream {
	return &TestStream{
		channel: make(chan *jasonsgame.UserInterfaceMessage, 25),
	}
}

func (ts *TestStream) Send(msg *jasonsgame.UserInterfaceMessage) error {
	ts.messages = append(ts.messages, msg)
	ts.channel <- msg
	return nil
}

func (ts *TestStream) GetMessages() []*jasonsgame.UserInterfaceMessage {
	return ts.messages
}

func (ts *TestStream) ClearMessages() error {
	ts.messages = NewTestStream().messages
	return nil
}

func (ts *TestStream) Channel() chan *jasonsgame.UserInterfaceMessage {
	return ts.channel
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

type UIInviteServer struct {
	network   network.Network
	stream    remoteStream
	doneChan  doneChan
	inkFaucet ink.Faucet
	invites   *actor.PID
}

func NewUIInviteProps(stream remoteStream, net network.Network, inkFaucet ink.Faucet) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &UIInviteServer{
			network:   net,
			stream:    stream,
			inkFaucet: inkFaucet,
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

func (uis *UIInviteServer) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		invitesActor := invites.NewInvitesActor(context.TODO(), invites.InvitesActorConfig{
			InkFaucet: uis.inkFaucet,
			Net:       uis.network,
		})
		actorCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
			return invitesActor
		}))
		uis.invites = invitesActor.PID()
		actorCtx.Send(actorCtx.Self(), &jasonsgame.MessageToUser{Message: "Welcome to Jason's Game! Please enter your invite code."})

	case *actor.Stopping:
		if uis.doneChan != nil {
			uis.doneChan <- struct{}{}
		}
		actorCtx.Poison(uis.invites)

	case *SetStream:
		// free up the previous stream
		uis.sendDone()

		uis.stream = msg.Stream
		uis.doneChan = msg.DoneChan

	case *jasonsgame.MessageToUser:
		actorCtx.SetReceiveTimeout(5 * time.Second)
		log.Debugf("message to user: %s", msg.Message)
		if uis.stream == nil {
			log.Errorf("no valid stream for user message: %v", msg.Message)
			return
		}

		uiMsg, err := buildUIMessage(msg)
		if err != nil {
			panic(err)
		}

		err = uis.stream.Send(uiMsg)
		if err != nil {
			uis.stream = nil
			uis.sendDone()
			log.Errorf("error sending message to stream: %v", err)
		}

	case *jasonsgame.UserInput:
		inviteSubmission := &inkfaucet.InviteSubmission{
			Invite: msg.Message,
		}

		req := actorCtx.RequestFuture(uis.invites, inviteSubmission, 10 * time.Second)

		uncastInviteResp, err := req.Result()
		if err != nil {
			panic("invalid invite code")
		}

		inviteResp, ok := uncastInviteResp.(*inkfaucet.InviteSubmissionResponse)
		if !ok {
			panic("invalid invite code")
		}

		if inviteResp.GetError() != "" {
			panic("invalid invite code")
		}

		actorCtx.Send(actorCtx.Self(), &jasonsgame.MessageToUser{Message: "Invite code accepted."})

		// TODO: Spawn a normal UIServer here, hand the stream and network over to it, & shut this actor down.
	}
}

func (uis *UIInviteServer) sendDone() {
	if uis.doneChan != nil {
		select {
		case uis.doneChan <- struct{}{}:
			log.Debugf("sent done")
		default:
			log.Warningf("nothing listening on done channel")
		}
	}
}
