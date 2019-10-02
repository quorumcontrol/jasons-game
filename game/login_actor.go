package game

import (
	"strings"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-datastore"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
)

var keystoreKey = datastore.NewKey("keystore-private-key")

type Login struct {
	game *Game
	cmds []string
}

func NewLogin(game *Game) *Login {
	return &Login{game: game, cmds: []string{}}
}

func (l *Login) HasState() bool {
	hasState, err := l.game.ds.Has(keystoreKey)
	if err != nil {
		panic(err)
	}
	return hasState
}

func (l *Login) NewActorProps() *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return l
	})
}

func (l *Login) commandUpdate(actorCtx actor.Context) {
	cmdUpdate := &jasonsgame.CommandUpdate{Commands: append(l.cmds, "help")}
	actorCtx.Send(l.game.ui, cmdUpdate)
}

func (l *Login) Receive(actorCtx actor.Context) {
	log.Debugf("login received: %+v", actorCtx.Message())

	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		actorCtx.Send(l.game.ui, &ui.SetGame{Game: actorCtx.Self()})

		var prompt string
		if l.HasState() {
			prompt = "Please enter your passsword:"
			l.cmds = []string{"recover"}
		} else {
			prompt = "Welcome to Jason's Game! Please type `sign up` or `recover` followed by your email to continue."
			l.cmds = []string{"sign up", "recover", "password:"}
		}

		l.game.sendUserMessage(actorCtx, prompt)
		l.commandUpdate(actorCtx)
	case *actor.Stopping:
		log.Info("login actor stopping")
	case *jasonsgame.UserInput:
		l.game.acknowledgeReceipt(actorCtx)

		cmdComponents := strings.Split(msg.Message, " ")
		switch cmd := cmdComponents[0]; cmd {
		case "password":
			l.game.sendUserMessage(actorCtx, "received "+msg.Message)
		case "sign up":
			if l.HasState() {
				panic("restart")
			}

			l.cmds = []string{"email:"}
			l.game.sendUserMessage(actorCtx, "please enter your email by type `email: `:")
		case "help":
			l.game.sendUserMessage(actorCtx, append(indentedList{"available commands:"}, l.cmds...))
		}

		log.Debugf("login actor received user input in: %+v", msg)
	case *jasonsgame.CommandUpdate:
		l.commandUpdate(actorCtx)
	case *ping:
		actorCtx.Respond(true)
	case *actor.Terminated:
		log.Info("login actor terminated")
	default:
		log.Debugf("login received: %v", msg)
	}
}
