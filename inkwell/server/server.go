package server

import (
	"context"
	"fmt"
	"os"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"

	iwconfig "github.com/quorumcontrol/jasons-game/inkwell/config"
	"github.com/quorumcontrol/jasons-game/inkwell/ink"
	"github.com/quorumcontrol/jasons-game/inkwell/inkwell"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("inkwell")

type InkwellRouter struct {
	parentCtx context.Context
	group     *types.NotaryGroup
	dataStore datastore.Batching
	net       network.Network
	inkwell   ink.Well
	tokenName *consensus.TokenName
	handler   *actor.PID
	inkActor  *actor.PID
}

func New(ctx context.Context, cfg iwconfig.InkwellConfig) (*InkwellRouter, error) {
	iw, err := iwconfig.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	inkDID := os.Getenv("INK_DID")

	if inkDID == "" {
		return nil, fmt.Errorf("INK_DID must be set")
	}

	sourceCfg := ink.ChainTreeInkwellConfig{
		Net: iw.Net,
	}

	ctiw, err := ink.NewChainTreeInkwell(sourceCfg)
	if err != nil {
		return nil, errors.Wrap(err, "error getting ink source")
	}

	return &InkwellRouter{
		parentCtx: ctx,
		group:     iw.NotaryGroup,
		dataStore: iw.DataStore,
		net:       iw.Net,
		inkwell:   ctiw,
		tokenName: &consensus.TokenName{ChainTreeDID: inkDID, LocalName: "ink"},
	}, nil
}

func (iw *InkwellRouter) Start() error {
	fmt.Println("starting inkwell service")

	arCtx := actor.EmptyRootContext

	act := arCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return iw
	}))
	iw.handler = act

	inkAct := ink.NewInkActor(iw.parentCtx, ink.InkActorConfig{
		Group:     iw.group,
		DataStore: iw.dataStore,
		Net:       iw.net,
		Inkwell:   iw.inkwell,
		TokenName: iw.tokenName,
	})

	inkAct.Start(arCtx)

	iw.inkActor = inkAct.PID()

	go func() {
		<-iw.parentCtx.Done()
		actor.EmptyRootContext.Stop(act)
	}()
	log.Info("serving ink & invite requests")

	// TODO: Subscribe to topic & listen for invite & ink requests

	return nil
}

func (iw *InkwellRouter) InkwellDID() string {
	return iw.inkwell.ChainTreeDID()
}

func (iw *InkwellRouter) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		// subscribe to community here?
	case *inkwell.InkRequest:
		log.Infof("Received InkRequest: %+v", msg)
		actorCtx.Forward(iw.inkActor)
	case *inkwell.InkResponse:
		log.Infof("Received InkResponse: %+v", msg)
		// Don't think these will come back to us (vs. original requestor)
	case *inkwell.InviteRequest:
		log.Infof("Received InviteRequest: %+v", msg)
	default:
		log.Warningf("Received unknown message type %T: %+v", msg, msg)
	}
}
