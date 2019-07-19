package invites

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/network"

	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
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

func (i *InvitesActor) PID() *actor.PID {
	return i.handler
}

func (i *InvitesActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		log.Info("invites actor started")
	case *inkfaucet.InviteRequest:
		log.Info("invites actor received invite request")
		i.handleInviteRequest(actorCtx)
	case *inkfaucet.InviteSubmission:
		log.Info("invites actor received invite submission")
		i.handleInviteSubmission(actorCtx)
	default:
		log.Warningf("invites actor received unknown message type %T: %+v", msg, msg)
	}
}

func (i *InvitesActor) handleInviteSubmission(actorCtx actor.Context) {
	// TODO: See if private key in submission matches a chaintree,
	//  create a new player chaintree if so,
	//  change owner of invite CT to new player,
	//  send all ink in invite CT to new player CT,
	//  delete invite CT.
}
