package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	ipfsconfig "github.com/ipfs/go-ipfs-config"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"

	"github.com/ipfs/go-datastore"
)

var addrFilters = []string{
	"10.0.0.0/8",
	"100.64.0.0/10",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.0.0/29",
	"192.0.0.8/32",
	"192.0.0.170/32",
	"192.0.0.171/32",
	"192.0.2.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"240.0.0.0/4",
}

var IpfsBootstrappers = append(ipfsconfig.DefaultBootstrapAddresses)

// var DefaultBootstrappers = []string{scalewayPeer}

func NewIPLDClient(ctx context.Context, key *ecdsa.PrivateKey, ds datastore.Batching) (*p2p.LibP2PHost, *p2p.BitswapPeer, error) {
	h, bitPeer, err := p2p.NewHostAndBitSwapPeer(
		ctx,
		p2p.WithListenIP("0.0.0.0", 0),
		p2p.WithKey(key),
		p2p.WithDatastore(ds),
		p2p.WithAutoRelay(true),
		p2p.WithRelayOpts(circuit.OptHop),
		p2p.WithDiscoveryNamespaces("jasons-game-tupelo"),
		p2p.WithAddrFilters(addrFilters),
	)
	log.Infof("started bitswapper %s", h.Identity())
	if err != nil {
		return nil, nil, fmt.Errorf("error creating hosts: %v", err)
	}
	return h, bitPeer, err
}
