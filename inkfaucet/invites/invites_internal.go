// +build internal

package invites

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/messages/build/go/transactions"

	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
)

const inviteInkAmount = uint64(1000)

func (i *InvitesActor) handleInviteRequest(actorCtx actor.Context) {
	inviteChainTree, inviteKey, err := i.net.CreateEphemeralChainTree()
	if err != nil {
		i.errorResponse(actorCtx, err, "error creating invite chaintree")
		return
	}

	log.Debugf("invite actor created ephemeral chaintree: %+v", *inviteChainTree)

	inkReq := &inkfaucet.InkRequest{
		Amount:             inviteInkAmount,
		DestinationChainId: inviteChainTree.MustId(),
	}

	log.Debugf("invite actor ink request: %+v", *inkReq)

	inviteInkReq := actorCtx.RequestFuture(i.inkActor.PID(), inkReq, 30 * time.Second)

	uncastInkResp, err := inviteInkReq.Result()
	if err != nil {
		i.errorResponse(actorCtx, err, "error getting ink for invite")
		return
	}

	log.Debugf("invite actor ink response: %+v", uncastInkResp)

	inkResp, ok := uncastInkResp.(*inkfaucet.InkResponse)
	if !ok {
		i.errorResponse(actorCtx, errors.Errorf("error casting ink response of type %T", uncastInkResp), "")
		return
	}

	var tokenPayload transactions.TokenPayload
	err = proto.Unmarshal(inkResp.Token, &tokenPayload)
	if err != nil {
		i.errorResponse(actorCtx, err, "error unmarshalling ink token payload")
		return
	}

	log.Debugf("invite actor unmarshalled token payload: %+v", tokenPayload)

	err = i.net.ReceiveInkOnEphemeralChainTree(inviteChainTree, inviteKey, &tokenPayload)
	if err != nil {
		i.errorResponse(actorCtx, err, "error receiving ink on invite chaintree")
		return
	}

	log.Debugf("invite actor received ink to ephemeral chaintree")

	serializedKey := crypto.FromECDSA(inviteKey)
	encodedKey := base58.Encode(serializedKey)

	actorCtx.Respond(&inkfaucet.InviteResponse{
		Invite: encodedKey,
	})
}

func (i *InvitesActor) errorResponse(actorCtx actor.Context, err error, msg string) {
	if msg != "" {
		err = errors.Wrap(err, msg)
	}
	log.Error(err)
	actorCtx.Respond(&inkfaucet.InviteResponse{
		Error: err.Error(),
	})
}
