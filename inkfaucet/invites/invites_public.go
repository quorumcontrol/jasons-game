// +build !internal

package invites

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"

	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("invites")

type InvitesActor struct {}

type InvitesActorConfig struct {
	InkFaucet *actor.PID
	Net       network.Network
}

func NewInvitesActor(ctx context.Context, cfg InvitesActorConfig) *InvitesActor {
	log.Error("invites are not available in public builds")
	return &InvitesActor{}
}

func (i *InvitesActor) Start(arCtx *actor.RootContext) {
	log.Error("invites are not available in public builds")
	// not available in public builds
}

func (i *InvitesActor) PID() *actor.PID {
	log.Error("invites are not available in public builds")
	// not available in public builds
	return nil
}
