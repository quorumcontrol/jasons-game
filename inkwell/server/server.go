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

	"github.com/quorumcontrol/jasons-game/config"
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
	inkSource ink.Source
	tokenName *consensus.TokenName
	handler   *actor.PID
}

type InkwellConfig struct {
	Local    bool
	S3Region string
	S3Bucket string
}

func NewServer(ctx context.Context, cfg InkwellConfig) (*InkwellRouter, error) {
	group, err := network.SetupTupeloNotaryGroup(ctx, cfg.Local)
	if err != nil {
		return nil, errors.Wrap(err,"error setting up notary group")
	}

	ds, err := config.S3DataStore(cfg.Local, cfg.S3Region, cfg.S3Bucket)
	if err != nil {
		panic(errors.Wrap(err, "error getting S3 data store"))
	}

	net, err := network.NewRemoteNetwork(ctx, group, ds)
	if err != nil {
		panic(errors.Wrap(err, "error setting up remote network"))
	}

	inkDID := os.Getenv("INK_DID")

	if inkDID == "" {
		panic("INK_DID must be set")
	}

	sourceCfg := ink.ChainTreeInkSourceConfig{
		Net: net,
	}

	inkSource, err := ink.NewChainTreeInkSource(sourceCfg)
	if err != nil {
		panic(errors.Wrap(err, "error getting ink source"))
	}

	return &InkwellRouter{
		parentCtx: ctx,
		group:     group,
		dataStore: ds,
		net:       net,
		inkSource: inkSource,
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
		InkSource: iw.inkSource,
		TokenName: iw.tokenName,
	})

	inkAct.Start(arCtx)

	go func() {
		<-iw.parentCtx.Done()
		actor.EmptyRootContext.Stop(act)
	}()
	log.Info("serving ink & invite requests")

	// TODO: Subscribe to topic & listen for invite & ink requests

	return nil
}

func (iw *InkwellRouter) Receive(aCtx actor.Context) {
	switch msg := aCtx.Message().(type) {
	case *actor.Started:
		// ignore
	case *inkwell.InkRequest:
		log.Infof("Received InkRequest: %+v", msg)
	case *inkwell.InkResponse:
		log.Infof("Received InkResponse: %+v", msg)
	case *inkwell.InviteRequest:
		log.Infof("Received InviteRequest: %+v", msg)
	default:
		log.Warningf("Received unknown message type %T: %+v", msg, msg)
	}
}