package provider

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"

	logging "github.com/ipfs/go-log"
	ifconnmgr "github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/pkg/errors"
	communitycfg "github.com/quorumcontrol/community/config"
	communityhub "github.com/quorumcontrol/community/hub"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"

	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("jasonblocks")

// Provider is a service that replaces an IPFS node as a bootstrapper
// it listens to some default topics, and provides a service where it will try to do a
// DAG on any CID sent to the BlockTopic which should cache it and make it available
// to any connected nodes.
type Provider struct {
	p2pHost           *p2p.LibP2PHost
	swapper           *p2p.BitswapPeer
	pubsubSystem      remote.PubSub
	handler           *actor.PID
	connectionManager ifconnmgr.ConnManager
	parentCtx         context.Context
	communityHub      *communityhub.Hub
}

const minConnections = 4915 // 60% of 8192 ulimit
const maxConnections = 7372 // 90% of 8192 ulimit
const connectionGracePeriod = 20 * time.Second

func New(ctx context.Context, key *ecdsa.PrivateKey, ds datastore.Batching, addlopts ...p2p.Option) (*Provider, error) {
	cm := connmgr.NewConnManager(minConnections, maxConnections, connectionGracePeriod)
	opts := append([]p2p.Option{
		p2p.WithLibp2pOptions(libp2p.ConnectionManager(cm)),
	}, addlopts...)
	host, peer, err := network.NewIPLDClient(ctx, key, ds, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "error creating ipld client")
	}

	pubsubSystem := remote.NewNetworkPubSub(host)

	communityName := "jasonsgame"
	hubConfig := &communitycfg.HubConfig{
		ClientConfig: &communitycfg.ClientConfig{
			Name:   communityName,
			Shards: 1024,
			PubSub: host.GetPubSub(),
		},
		CacheMessages: false,
	}

	return &Provider{
		p2pHost:           host,
		swapper:           peer,
		pubsubSystem:      pubsubSystem,
		parentCtx:         ctx,
		connectionManager: cm,
		communityHub:      communityhub.New(ctx, hubConfig),
	}, nil
}

func (p *Provider) Start() error {
	fmt.Printf("starting %s\naddresses:%v\n", p.p2pHost.Identity(), p.p2pHost.Addresses())

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := p.p2pHost.Bootstrap(network.IpfsBootstrappers)
		if err != nil {
			log.Errorf("error bootstrapping ipld host: %v", err)
		}
	}()

	wg.Add(1)
	func() {
		defer wg.Done()
		_, err := p.p2pHost.Bootstrap(network.GameBootstrappers())
		if err != nil {
			log.Errorf("error bootstrapping ipld host: %v", err)
		}
	}()
	wg.Wait()

	err := p.communityHub.Start()
	if err != nil {
		return errors.Wrap(err, "could not start community hub")
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
		actor.EmptyRootContext.Stop(act)
	}()
	log.Infof("serving a provider now")

	return nil

}
