package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	libp2pcrypto "github.com/libp2p/go-libp2p-crypto"

	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	ipfslite "github.com/quorumcontrol/jasons-game/ipfslite"
)

func NewIPLDClient(ctx context.Context, key *ecdsa.PrivateKey, ds datastore.Batching) (*ipfslite.Peer, error) {

	priv, err := p2pPrivateFromEcdsaPrivate(key)
	if err != nil {
		return nil, errors.Wrap(err, "error getting private key from key")
	}

	listen, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")

	h, dht, err := ipfslite.SetupLibp2p(
		ctx,
		priv,
		[]multiaddr.Multiaddr{listen},
		nil, // no bootstrap
	)

	if err != nil {
		panic(err)
	}

	lite, err := ipfslite.New(ctx, ds, h, dht)
	if err != nil {
		panic(err)
	}

	disoverer := newJasonsDiscoverer(h, dht)
	go func() {
		err := disoverer.doDiscovery(ctx)
		if err != nil {
			log.Errorf("error doing discovery: %v", err)
		}
	}()

	return lite, nil
}

const expectedKeySize = 32

func p2pPrivateFromEcdsaPrivate(key *ecdsa.PrivateKey) (libp2pcrypto.PrivKey, error) {
	// private keys can be 31 or 32 bytes for ecdsa.PrivateKey, but must be 32 Bytes for libp2pcrypto,
	// so we zero pad the slice if it is 31 bytes.
	keyBytes := key.D.Bytes()
	if (len(keyBytes) != expectedKeySize) && (len(keyBytes) != (expectedKeySize - 1)) {
		return nil, fmt.Errorf("error: length of private key must be 31 or 32 bytes")
	}
	keyBytes = append(make([]byte, expectedKeySize-len(keyBytes)), keyBytes...)
	libp2pKey, err := libp2pcrypto.UnmarshalSecp256k1PrivateKey(keyBytes)
	if err != nil {
		return libp2pKey, fmt.Errorf("error unmarshaling: %v", err)
	}
	return libp2pKey, err
}
