// +build integration

package network

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/stretchr/testify/require"
)

func TestCreateChainTree(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := SetupTupeloNotaryGroup(ctx, true)
	require.Nil(t, err)

	net, err := NewRemoteNetworkWithConfig(ctx, &RemoteNetworkConfig{
		NotaryGroup: group,
		KeyValueStore: config.MemoryDataStore(),
	})
	require.Nil(t, err)
	require.NotNil(t, net)

	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	_, err = net.Tupelo.CreateChainTree(key)
	require.Nil(t, err)
}

func TestGetTip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := SetupTupeloNotaryGroup(ctx, true)
	require.Nil(t, err)

	net, err := NewRemoteNetworkWithConfig(ctx, &RemoteNetworkConfig{
		NotaryGroup: group,
		KeyValueStore: config.MemoryDataStore(),
	})
	require.Nil(t, err)
	require.NotNil(t, net)
	tup := net.Tupelo

	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	tree, err := tup.CreateChainTree(key)
	require.Nil(t, err)
	err = tup.UpdateChainTree(tree, key, "foo", "bar")
	require.Nil(t, err)

	time.Sleep(100 * time.Millisecond)
	tip, err := tup.GetTip(tree.MustId())
	require.Nil(t, err)

	require.Equal(t, tree.Tip(), tip)
}
