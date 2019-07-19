// +build internal

package invites

import (
	"context"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/messages/build/go/transactions"

	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("invites")

type InvitesActor struct {
	parentCtx context.Context
	handler   *actor.PID
	inkFaucet *actor.PID
	net       network.Network
}

type InvitesActorConfig struct {
	InkFaucet *actor.PID
	Net       network.Network
}

func NewInvitesActor(ctx context.Context, cfg InvitesActorConfig) *InvitesActor {
	return &InvitesActor{
		parentCtx: ctx,
		inkFaucet: cfg.InkFaucet,
		net:       cfg.Net,
	}
}

func (i *InvitesActor) Start(arCtx *actor.RootContext) {
	act := arCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return i
	}))

	i.handler = act

	go func() {
		<-i.parentCtx.Done()
		arCtx.Stop(act)
	}()
}

func (i *InvitesActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		log.Info("invites actor started")
	case *inkfaucet.InviteRequest:
		log.Info("invites actor received invite request")
		i.handleInviteRequest(actorCtx)
	default:
		log.Warningf("invites actor received unknown message type %T: %+v", msg, msg)
	}
}

func (i *InvitesActor) PID() *actor.PID {
	return i.handler
}

func (i *InvitesActor) handleInviteRequest(actorCtx actor.Context) {
	inviteChainTree, inviteKey, err := i.net.CreateEphemeralChainTree()
	if err != nil {
		i.errorResponse(actorCtx, err, "error creating invite chaintree")
		return
	}

	log.Debugf("invite actor created ephemeral chaintree: %+v", *inviteChainTree)

	inkReq := &inkfaucet.InkRequest{
		Amount:             1,
		DestinationChainId: inviteChainTree.MustId(),
	}

	log.Debugf("invite actor ink request: %+v", *inkReq)

	inviteInkReq := actorCtx.RequestFuture(i.inkFaucet, inkReq, 30 * time.Second)

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

	actorCtx.Respond(&inkfaucet.InviteResponse{
		Invite: base58.Encode(crypto.FromECDSA(inviteKey)),
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
