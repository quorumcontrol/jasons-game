package invites

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"

	"github.com/quorumcontrol/jasons-game/inkwell/inkwell"
)

var log = logging.Logger("invites")

type InvitesActor struct {
	inkWell *actor.PID
}

func (i *InvitesActor) Receive(aCtx actor.Context) {
	switch msg := aCtx.Message().(type) {
	case *actor.Started:
		log.Info("invites actor started")
	case *inkwell.InviteRequest:
		log.Infof("invites actor received invite request: %+v", msg)
	default:
		log.Warningf("invites actor received unknown message type %T: %+v", msg, msg)
	}
}
