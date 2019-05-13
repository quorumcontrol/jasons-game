package provider

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

var log = logging.Logger("jasonblocks")

const BlockTopic = "jasons-game-tupelo-world-blocks"
const ShoutTopic = "jasons-game-shouting-players"

var topics = []string{
	BlockTopic,
	ShoutTopic,
}

// Provider is a service that replaces an IPFS node as a bootstrapper
// it listens to some default topics, and provides a service where it will try to do a
// DAG on any CID sent to the BlockTopic which should cache it and make it available
// to any connected nodes.
type Provider struct {
	p2pHost      *p2p.LibP2PHost
	swapper      *p2p.BitswapPeer
	pubsubSystem remote.PubSub
	handler      *actor.PID
	parentCtx    context.Context
}

func New(ctx context.Context, key *ecdsa.PrivateKey, ds datastore.Batching, opts ...p2p.Option) (*Provider, error) {
	host, peer, err := network.NewIPLDClient(ctx, key, ds, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "error creating ipld client")
	}

	pubsubSystem := remote.NewNetworkPubSub(host)

	return &Provider{
		p2pHost:      host,
		swapper:      peer,
		pubsubSystem: pubsubSystem,
		parentCtx:    ctx,
	}, nil
}

func (p *Provider) Start() error {
	fmt.Printf("starting %s\naddresseses:%v\n", p.p2pHost.Identity(), p.p2pHost.Addresses())

	_, err := p.p2pHost.Bootstrap(network.IpfsBootstrappers)
	if err != nil {
		log.Errorf("error bootstrapping ipld host: %v", err)
	}

	//TODO: once there's more than one:
	// _, err = p.p2pHost.Bootstrap(network.GameBootstrappers)
	// if err != nil {
	// 	log.Errorf("error bootstrapping ipld host: %v", err)
	// }

	// subscribe with a noop to shouting - so that we forward it through
	sub, err := p.p2pHost.GetPubSub().Subscribe(ShoutTopic)
	if err != nil {
		return errors.Wrap(err, "error subscriging to ShoutTopic")
	}

	// subscribe with the block getter
	act := actor.EmptyRootContext.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return &distributer{
			provider: p,
		}
	}))
	p.handler = act

	go func() {
		<-p.parentCtx.Done()
		sub.Cancel()
		act.Stop()
	}()
	log.Infof("serving a provider now")

	return nil

}
