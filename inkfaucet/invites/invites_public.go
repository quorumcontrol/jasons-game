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
	// not available in public builds
	return &InvitesActor{}
}

func (i *InvitesActor) Start(arCtx *actor.RootContext) {
	// not available in public builds
}

func (i *InvitesActor) PID() *actor.PID {
	// not available in public builds
	return nil
}
