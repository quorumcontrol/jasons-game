package server

import (
	"context"
	"crypto/ecdsa"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/inkfaucet/config"
	"github.com/quorumcontrol/jasons-game/inkfaucet/ink"
	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
	"github.com/quorumcontrol/jasons-game/inkfaucet/invites"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("inkFaucetRouter")

type InkFaucetRouter struct {
	parentCtx    context.Context
	net          network.Network
	inkFaucet    ink.Faucet
	tokenName    *consensus.TokenName
	handler      *actor.PID
	inkActor     *ink.InkActor
	invitesActor *actor.PID
}

func KeyToDID(key *ecdsa.PrivateKey) string {
	return consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
}

func New(ctx context.Context, cfg config.InkFaucetConfig) (*InkFaucetRouter, error) {
	iw, err := config.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	faucetConfig := ink.ChainTreeInkFaucetConfig{
		Net:         iw.Net,
		InkOwnerDID: cfg.InkOwnerDID,
		PrivateKey:  cfg.PrivateKey,
	}

	ctif, err := ink.NewChainTreeInkFaucet(faucetConfig)
	if err != nil {
		return nil, errors.Wrap(err, "error creating ink faucet")
	}

	return &InkFaucetRouter{
		parentCtx: ctx,
		net:       iw.Net,
		inkFaucet: ctif,
		tokenName: &consensus.TokenName{ChainTreeDID: cfg.InkOwnerDID, LocalName: "ink"},
	}, nil
}

func (ifr *InkFaucetRouter) Start(allowInvites bool) error {
	log.Info("starting inkFaucet service")

	arCtx := actor.EmptyRootContext

	act := arCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return ifr
	}))
	ifr.handler = act

	inkAct := ink.NewInkActor(ifr.parentCtx, ink.InkActorConfig{
		InkFaucet: ifr.inkFaucet,
	})

	inkAct.Start(arCtx)

	ifr.inkActor = inkAct

	if allowInvites {
		invitesActor := invites.NewInvitesActor(ifr.parentCtx, invites.InvitesActorConfig{
			InkActor: ifr.inkActor,
			Net:      ifr.net,
		})

		invitesActor.Start(arCtx)

		ifr.invitesActor = invitesActor.PID()
	}

	go func() {
		<-ifr.parentCtx.Done()
		arCtx.Stop(act)
	}()

	if allowInvites {
		log.Info("serving ink & invite requests")
	} else {
		log.Info("serving ink requests")
	}

	return nil
}

func (ifr *InkFaucetRouter) InkFaucetDID() string {
	return ifr.inkFaucet.ChainTreeDID()
}

func (ifr *InkFaucetRouter) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
	case *inkfaucet.InkRequest:
		log.Infof("Received InkRequest: %+v", msg)
		actorCtx.Forward(ifr.inkActor.PID())
	case *inkfaucet.InviteRequest:
		log.Infof("Received InviteRequest: %+v", msg)
		actorCtx.Forward(ifr.invitesActor)
	}
}

func (ifr *InkFaucetRouter) PID() *actor.PID {
	return ifr.handler
}
