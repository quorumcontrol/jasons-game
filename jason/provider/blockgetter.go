package provider

import (
	"context"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/router"
	"github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/jasons-game/network"
)

const getterConcurrency = 50

var getTimeout = 20 * time.Second

type distributer struct {
	provider *Provider
	pool     *actor.PID
}

func (d *distributer) Receive(aCtx actor.Context) {
	switch msg := aCtx.Message().(type) {
	case *actor.Started:
		aCtx.Spawn(d.provider.pubsubSystem.NewSubscriberProps(network.BlockTopic))
		d.pool = aCtx.Spawn(router.NewRoundRobinPool(getterConcurrency).WithProducer(func() actor.Actor {
			return &getter{
				provider: d.provider,
			}
		}))
	case *network.Block:
		aCtx.Forward(d.pool)
	case *network.Join:
		peerID, err := peer.IDB58Decode(msg.Identity)
		if err != nil {
			log.Errorf("received invalid join message with identity %s", msg.Identity)
			return
		}
		d.provider.connectionManager.Protect(peerID, "player")
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

		if len(msg.Data) > 0 {
			log.Debugf("received block with data %s", msg.Cid)
			// if we have data, we can just add it right away.
			sw := &safewrap.SafeWrap{}
			n := sw.Decode(msg.Data)
			if sw.Err != nil {
				log.Errorf("error decoding block: %v", sw.Err)
				return
			}
			// if the Block comes in with Data attached, then we can just add it directly to the provider
			err := g.provider.swapper.Add(ctx, n)
			if err != nil {
				log.Errorf("error adding block: %v", err)
				return
			}
			return
		}
		// otherwise, we need to get the block by CID.

		// this will have CID, and we just do a get on the block
		id, err := cid.Cast(msg.Cid)
		if err != nil {
			log.Errorf("error casting CID: %v", err)
		}
		log.Debugf("fetching %s", id.String())
		_, err = g.provider.swapper.Get(ctx, id)
		if err != nil {
			log.Errorf("error getting block: %v", err)
			return
		}
		log.Debugf("success %s", id.String())
	}
}
