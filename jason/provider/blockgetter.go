package provider

import (
	"context"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/router"
	"github.com/ipfs/go-cid"
	"github.com/quorumcontrol/jasons-game/network"
)

const getterConcurrency = 50

var getTimeout = 5 * time.Second

type distributer struct {
	provider *Provider
	pool     *actor.PID
}

func (d *distributer) Receive(aCtx actor.Context) {
	switch aCtx.Message().(type) {
	case *actor.Started:
		aCtx.Spawn(d.provider.pubsubSystem.NewSubscriberProps(network.BlockTopic))
		d.pool = aCtx.Spawn(router.NewRoundRobinPool(getterConcurrency).WithProducer(func() actor.Actor {
			return &getter{
				provider: d.provider,
			}
		}))
	case *network.Block:
		aCtx.Forward(d.pool)
	}
}

type getter struct {
	provider *Provider
}

func (g *getter) Receive(aCtx actor.Context) {
	switch msg := aCtx.Message().(type) {
	case *network.Block:
		ctx, cancel := context.WithTimeout(context.Background(), getTimeout)
		defer cancel()
		// this will have CID, and we just do a get on the block
		id, err := cid.Cast(msg.Cid)
		if err != nil {
			log.Errorf("error casting CID: %v", err)
		}
		log.Debugf("fetching %s", id.String())
		_, err = g.provider.swapper.Get(ctx, id)
		if err != nil {
			log.Errorf("error getting block: %v", err)
		}
	}
}
