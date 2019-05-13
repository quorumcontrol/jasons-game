package provider

import (
	"context"
	"testing"

	logging "github.com/ipfs/go-log"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	logging.SetLogLevel("*", "info")
	logging.SetLogLevel("tupelop2p", "debug")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	ds := datastore.NewMapDatastore()

	provider, err := New(ctx, key, ds)
	require.Nil(t, err)

	err = provider.Start()
	require.Nil(t, err)
}
