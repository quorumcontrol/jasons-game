package server

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"

	iwconfig "github.com/quorumcontrol/jasons-game/inkfaucet/config"
	"github.com/quorumcontrol/jasons-game/inkfaucet/ink"
	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("inkfaucet")

type InkFaucetRouter struct {
	parentCtx context.Context
	group     *types.NotaryGroup
	dataStore datastore.Batching
	net       network.Network
	inkfaucet   ink.Well
	tokenName *consensus.TokenName
	handler   *actor.PID
	inkActor  *actor.PID
}

func New(ctx context.Context, cfg iwconfig.InkFaucetConfig) (*InkFaucetRouter, error) {
	iw, err := iwconfig.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	sourceCfg := ink.ChainTreeInkFaucetConfig{
		Net:         iw.Net,
		InkOwnerDID: cfg.InkOwnerDID,
	}

	ctiw, err := ink.NewChainTreeInkFaucet(sourceCfg)
	if err != nil {
		return nil, errors.Wrap(err, "error getting ink source")
	}

	return &InkFaucetRouter{
		parentCtx: ctx,
		group:     iw.NotaryGroup,
		dataStore: iw.DataStore,
		net:       iw.Net,
		inkfaucet:   ctiw,
		tokenName: &consensus.TokenName{ChainTreeDID: cfg.InkOwnerDID, LocalName: "ink"},
	}, nil
}

func (iw *InkFaucetRouter) Start() error {
	log.Info("starting inkfaucet service")

	arCtx := actor.EmptyRootContext

	act := arCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return iw
	}))
	iw.handler = act

	inkAct := ink.NewInkActor(iw.parentCtx, ink.InkActorConfig{
		Group:     iw.group,
		DataStore: iw.dataStore,
		Net:       iw.net,
		InkFaucet:   iw.inkfaucet,
		TokenName: iw.tokenName,
	})

	inkAct.Start(arCtx)

	iw.inkActor = inkAct.PID()

	go func() {
		<-iw.parentCtx.Done()
		arCtx.Stop(act)
	}()
	log.Info("serving ink & invite requests")

	// TODO: Subscribe to topic & listen for invite & ink requests

	return nil
}

func (iw *InkFaucetRouter) InkFaucetDID() string {
	return iw.inkfaucet.ChainTreeDID()
}

func (iw *InkFaucetRouter) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		// subscribe to community here?
	case *inkfaucet.InkRequest:
		log.Infof("Received InkRequest: %+v", msg)
		actorCtx.Forward(iw.inkActor)
	case *inkfaucet.InviteRequest:
		log.Infof("Received InviteRequest: %+v", msg)
		// TODO: Write me
	}
}
