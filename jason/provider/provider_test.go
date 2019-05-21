package provider

import (
	"context"
	"strings"
	"testing"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/require"
)

func newDatastore() datastore.Batching {
	return dssync.MutexWrap(datastore.NewMapDatastore())
}

func TestStart(t *testing.T) {
	logging.SetLogLevel("*", "info")
	logging.SetLogLevel("tupelop2p", "debug")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	ds := newDatastore()

	provider, err := New(ctx, key, ds)
	require.Nil(t, err)

	err = provider.Start()
	require.Nil(t, err)
}

func TestProviderPubsubRelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	ds := newDatastore()

	provider, err := New(ctx, key, ds)
	require.Nil(t, err)

	err = provider.Start()
	require.Nil(t, err)

	keyA, err := crypto.GenerateKey()
	require.Nil(t, err)

	keyB, err := crypto.GenerateKey()
	require.Nil(t, err)

	storeA := datastore.NewMapDatastore()
	storeB := datastore.NewMapDatastore()

	hosta, _, err := network.NewIPLDClient(ctx, keyA, storeA)
	require.Nil(t, err)

	hostb, _, err := network.NewIPLDClient(ctx, keyB, storeB)
	require.Nil(t, err)

	hosta.Bootstrap(bootstrapAddresses(provider.p2pHost))
	hostb.Bootstrap(bootstrapAddresses(provider.p2pHost))

	err = hosta.WaitForBootstrap(2, 5*time.Second)
	require.Nil(t, err)

	err = hostb.WaitForBootstrap(2, 5*time.Second)
	require.Nil(t, err)

	err = provider.p2pHost.WaitForBootstrap(2, 10*time.Second)
	require.Nil(t, err)
}

func bootstrapAddresses(bootstrapHost *p2p.LibP2PHost) []string {
	addresses := bootstrapHost.Addresses()
	for _, addr := range addresses {
		addrStr := addr.String()
		if strings.Contains(addrStr, "127.0.0.1") {
			return []string{addrStr}
		}
	}
	return nil
}
