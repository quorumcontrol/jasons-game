// +build !internal

package invites

import (
	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
)

func (i *InvitesActor) handleInviteRequest(actorCtx actor.Context) {
	actorCtx.Respond(&inkfaucet.InkResponse{
		Error: "invite requests not available in public builds",
	})
}
