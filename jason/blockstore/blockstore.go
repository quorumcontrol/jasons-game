package blockstore

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"time"

	datastore "github.com/ipfs/go-datastore"
	libp2p "github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"

	"github.com/gogo/protobuf/proto"
	logging "github.com/ipfs/go-log"
	ifconnmgr "github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/pkg/errors"
	communitycfg "github.com/quorumcontrol/community/config"
	communityhub "github.com/quorumcontrol/community/hub"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/quorumcontrol/community/pb/messages"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("jasonblocks")

const minConnections = 4915 // 60% of 8192 ulimit
const maxConnections = 7372 // 90% of 8192 ulimit
const connectionGracePeriod = 20 * time.Second

// Blockstore is a service that stores chaintree nodes and current
// states from tupelo broadcast, providing them back over ipfs bitswap
type Blockstore struct {
	p2pHost           *p2p.LibP2PHost
	swapper           *p2p.BitswapPeer
	connectionManager ifconnmgr.ConnManager
	parentCtx         context.Context
	communityHub      *communityhub.Hub
	jasonCommunity    *network.Community
}

func New(ctx context.Context, key *ecdsa.PrivateKey, ds datastore.Batching, ng *types.NotaryGroup, addlopts ...p2p.Option) (*Blockstore, error) {
	cm := connmgr.NewConnManager(minConnections, maxConnections, connectionGracePeriod)
	opts := append([]p2p.Option{
		p2p.WithLibp2pOptions(libp2p.ConnectionManager(cm)),
	}, addlopts...)
	host, bitswap, err := network.NewIPLDClient(ctx, key, ds, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "error creating ipld client")
	}

	hubConfig := &communitycfg.HubConfig{
		ClientConfig: &communitycfg.ClientConfig{
			Name:   network.CommunityName,
			Shards: 1024,
			PubSub: host.GetPubSub(),
		},
		CacheMessages: false,
		CacheBlocks:   true,
		Datastore:     ds,
		Dagstore:      bitswap,
		NotaryGroup:   ng,
	}

	return &Blockstore{
		p2pHost:           host,
		swapper:           bitswap,
		parentCtx:         ctx,
		connectionManager: cm,
		communityHub:      communityhub.New(ctx, hubConfig),
		jasonCommunity:    network.NewJasonCommunity(ctx, key, host),
	}, nil
}

func (p *Blockstore) Start() error {
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := p.p2pHost.Bootstrap(network.IpfsBootstrappers)
		if err != nil {
			log.Errorf("error bootstrapping ipld host for ipfs: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := p.p2pHost.Bootstrap(network.GameBootstrappers()); err != nil {
			log.Errorf("error bootstrapping ipld host for jason: %v", err)
			return
		}
		if err := p.p2pHost.WaitForBootstrap(1, 1*time.Second); err != nil {
			log.Errorf("error waiting to bootstrap on ipld host for jason: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		bootstrappers, err := network.TupeloBootstrappers()

		if err != nil {
			log.Errorf("error reading bootstrap addresses for tupelo: %v", err)
			return
		}

		if _, err := p.p2pHost.Bootstrap(bootstrappers); err != nil {
			log.Errorf("error bootstrapping ipld host for tupelo: %v", err)
			return
		}
		if err := p.p2pHost.WaitForBootstrap(1, 1*time.Second); err != nil {
			log.Errorf("error waiting to bootstrap on ipld host for tupelo: %v", err)
		}
	}()
	wg.Wait()

	err := p.communityHub.Start()
	if err != nil {
		return errors.Wrap(err, "could not start community hub")
	}

	log.Infof("serving a provider now")

	_, err = p.jasonCommunity.Subscribe([]byte(network.GeneralTopic), func(_ context.Context, _ *messages.Envelope, protoMsg proto.Message) {
		switch msg := protoMsg.(type) {
		case *network.Join:
			peerID, err := peer.IDB58Decode(msg.Identity)
			if err != nil {
				log.Errorf("received invalid join message with identity %s", msg.Identity)
				return
			}
			p.connectionManager.Protect(peerID, "player")
		}
	})
	if err != nil {
		return errors.Wrap(err, "error subscribing to community")
	}

	return nil
}
