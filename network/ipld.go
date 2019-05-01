package network

import (
	"context"

	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pnet "github.com/libp2p/go-libp2p-pnet"
	"github.com/multiformats/go-multiaddr"
	ipnet "github.com/libp2p/go-libp2p-interface-pnet"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"

)

func NewIPLDClient(ctx context.Context, ds datastore.Batching) (*ipfslite.Peer, error) {
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		panic(err)
	}

	listen, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/4005")

	h, dht, err := SetupLibp2p(
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

	go lite.Bootstrap(ipfslite.DefaultBootstrapPeers())

	disoverer := newJasonsDiscoverer(h, dht)
	go disoverer.doDiscovery(ctx)

	return lite, nil
}

func SetupLibp2p(
	ctx context.Context,
	hostKey crypto.PrivKey,
	secret []byte,
	listenAddrs []multiaddr.Multiaddr,
) (host.Host, *dht.IpfsDHT, error) {

	var prot ipnet.Protector
	var err error

	// Create protector if we have a secret.
	if secret != nil && len(secret) > 0 {
		var key [32]byte
		copy(key[:], secret)
		prot, err = pnet.NewV1ProtectorFromBytes(&key)
		if err != nil {
			return nil, nil, err
		}
	}

	h, err := libp2p.New(
		ctx,
		libp2p.Identity(hostKey),
		libp2p.ListenAddrs(listenAddrs...),
		libp2p.PrivateNetwork(prot),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(circuit.OptHop),
	)
	if err != nil {
		return nil, nil, err
	}

	idht, err := dht.New(ctx, h)
	if err != nil {
		h.Close()
		return nil, nil, err
	}

	rHost := routedhost.Wrap(h, idht)
	return rHost, idht, nil
}
