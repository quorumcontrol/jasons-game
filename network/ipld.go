package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/quorumcontrol/tupelo-go-sdk/p2p"

	"github.com/ipfs/go-datastore"
)

var addrFilters = []string{
	"10.0.0.0/8",
	"100.64.0.0/10",
	"169.254.0.0/16",
	// "172.16.0.0/12", // we use this for docker
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

// TODO just switch these two out with options builder and call the NewHosts directly
func NewIPLDClient(ctx context.Context, key *ecdsa.PrivateKey, ds datastore.Batching, addlOpts ...p2p.Option) (*p2p.LibP2PHost, *p2p.BitswapPeer, error) {
	opts := append([]p2p.Option{
		p2p.WithListenIP("0.0.0.0", 0),
		p2p.WithKey(key),
		p2p.WithDatastore(ds),
		p2p.WithAddrFilters(addrFilters),
	}, addlOpts...)

	h, bitPeer, err := p2p.NewHostAndBitSwapPeer(
		ctx,
		opts...,
	)
	log.Infof("started bitswapper %s", h.Identity())
	if err != nil {
		return nil, nil, fmt.Errorf("error creating hosts: %v", err)
	}
	return h, bitPeer, err
}

func NewLibP2PHost(ctx context.Context, key *ecdsa.PrivateKey, addlOpts ...p2p.Option) (*p2p.LibP2PHost, error) {
	opts := append([]p2p.Option{
		p2p.WithListenIP("0.0.0.0", 0),
		p2p.WithKey(key),
		p2p.WithAddrFilters(addrFilters),
	}, addlOpts...)

	h, err := p2p.NewHostFromOptions(ctx, opts...)
	log.Infof("started libp2p host %s", h.Identity())
	if err != nil {
		return nil, fmt.Errorf("error creating hosts: %v", err)
	}
	return h, err
}
