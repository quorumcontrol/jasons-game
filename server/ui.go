package server

import (
	"fmt"
	"log"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

type uiServer struct {
	game    *actor.PID
	network network.Network
	stream  jasonsgame.GameService_ReceiveUserMessagesServer
}

func NewUIProps(net network.Network) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &uiServer{
			network: net,
		}
	})
}

type subscribeStream struct {
	stream jasonsgame.GameService_ReceiveUserMessagesServer
}

func (us *uiServer) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		gameActor, err := actorCtx.SpawnNamed(game.NewGameProps(actorCtx.Self(), us.network), "game")
		if err != nil {
			panic(fmt.Errorf("error running UI: %v", err))
		}
		us.game = gameActor
	case subscribeStream:
		us.stream = msg.stream
	case *jasonsgame.MessageToUser:
		log.Printf("message to user: %s", msg.Message)
		if us.stream != nil {
			us.stream.Send(msg)
			return
		}
		log.Printf("no stream to send")

	case *jasonsgame.UserInput:
		log.Printf("user input %s", msg.Message)
		actorCtx.Send(us.game, msg)
	}
}
