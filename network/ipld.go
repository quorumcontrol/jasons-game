package network

import (
	"context"

	"github.com/ipfs/go-datastore"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multiaddr"
	ipfslite "github.com/quorumcontrol/jasons-game/ipfslite"
)

func NewIPLDClient(ctx context.Context, ds datastore.Batching) (*ipfslite.Peer, error) {
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		panic(err)
	}

	listen, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")

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

	log.Info("bootstraping")
	lite.Bootstrap(ipfslite.DefaultBootstrapPeers())

	disoverer := newJasonsDiscoverer(h, dht)
	go disoverer.doDiscovery(ctx)

	return lite, nil
}
