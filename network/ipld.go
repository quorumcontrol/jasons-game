package network

import (
	"context"

	ipfslite "github.com/hsanjuan/ipfs-lite"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multiaddr"
)

func NewIPLDClient(ctx context.Context, path string) (*ipfslite.Peer, error) {
	ds, err := ipfslite.BadgerDatastore(path)
	if err != nil {
		panic(err)
	}
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		panic(err)
	}

	listen, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/4005")

	h, dht, err := ipfslite.SetupLibp2p(
		ctx,
		priv,
		nil,
		[]multiaddr.Multiaddr{listen},
	)

	if err != nil {
		panic(err)
	}

	lite, err := ipfslite.New(ctx, ds, h, dht, nil)
	if err != nil {
		panic(err)
	}

	lite.Bootstrap(ipfslite.DefaultBootstrapPeers())

	disoverer := newJasonsDiscoverer(h, dht)
	disoverer.doDiscovery(ctx)

	return lite, nil
}
