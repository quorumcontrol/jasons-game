package network

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"

	"github.com/quorumcontrol/tupelo-go-sdk/p2p"

	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	mh "github.com/multiformats/go-multihash"
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

const scalewayPeer = "/ip4/51.158.189.66/tcp/4001/ipfs/QmSWp7tT6hBPAEvDEoz76axX3HHT87vyYN2vEMyiwmcFZk"

// var DefaultBootstrappers = append(ipfsconfig.DefaultBootstrapAddresses, scalewayPeer)
var DefaultBootstrappers = []string{scalewayPeer}

func NewIPLDClient(ctx context.Context, key *ecdsa.PrivateKey, ds datastore.Batching) (*p2p.LibP2PHost, *p2p.BitswapPeer, error) {
	cid, _ := nsToCid("jasons-game-tupelo")
	log.Infof("namespace CID: %s: base64: %s", cid.String(), base64.StdEncoding.EncodeToString([]byte("jasons-game-tupelo")))
	h, bitPeer, err := p2p.NewHostAndBitSwapPeer(
		ctx,
		p2p.WithListenIP("0.0.0.0", 0),
		p2p.WithKey(key),
		p2p.WithDatastore(ds),
		p2p.WithAutoRelay(true),
		p2p.WithDiscoveryNamespaces("jasons-game-tupelo"),
		p2p.WithAddrFilters(addrFilters),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating hosts: %v", err)
	}
	return h, bitPeer, err
}

func nsToCid(ns string) (cid.Cid, error) {
	h, err := mh.Sum([]byte(ns), mh.SHA2_256, -1)
	if err != nil {
		return cid.Undef, err
	}

	return cid.NewCidV1(cid.Raw, h), nil
}
