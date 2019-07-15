package bootstrap

import (
	"context"
	"crypto/ecdsa"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/pkg/errors"
	communitycfg "github.com/quorumcontrol/community/config"
	communityhub "github.com/quorumcontrol/community/hub"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"

	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("jasonbootstrap")

// Bootstrap is p2p host for games to bootstrap with
type Bootstrap struct {
	p2pHost      *p2p.LibP2PHost
	parentCtx    context.Context
	communityHub *communityhub.Hub
}

const minConnections = 4915 // 60% of 8192 ulimit
const maxConnections = 7372 // 90% of 8192 ulimit
const connectionGracePeriod = 20 * time.Second

func New(ctx context.Context, key *ecdsa.PrivateKey, addlopts ...p2p.Option) (*Bootstrap, error) {
	cm := connmgr.NewConnManager(minConnections, maxConnections, connectionGracePeriod)
	opts := append([]p2p.Option{
		p2p.WithLibp2pOptions(libp2p.ConnectionManager(cm)),
	}, addlopts...)

	host, err := network.NewLibP2PHost(ctx, key, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "error creating libp2p host")
	}

	hubConfig := &communitycfg.HubConfig{
		ClientConfig: &communitycfg.ClientConfig{
			Name:   network.CommunityName,
			Shards: 1024,
			PubSub: host.GetPubSub(),
		},
		CacheMessages: false,
	}

	return &Bootstrap{
		p2pHost:      host,
		parentCtx:    ctx,
		communityHub: communityhub.New(ctx, hubConfig),
	}, nil
}

func (b *Bootstrap) Start() error {
	var err error

	bootstrappers := network.GameBootstrappers()
	if len(bootstrappers) > 1 {
		_, err = b.p2pHost.Bootstrap(bootstrappers)
		if err != nil {
			log.Errorf("error bootstrapping ipld host: %v", err)
		}
	}

	err = b.communityHub.Start()
	if err != nil {
		return errors.Wrap(err, "could not start community hub")
	}

	log.Infof("bootstrap listening at: %v", b.p2pHost.Addresses())

	return nil
}
