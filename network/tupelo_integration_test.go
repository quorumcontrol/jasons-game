// +build integration

package network

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/stretchr/testify/require"
)

func setupRemote(ctx context.Context, group *types.NotaryGroup) (p2p.Node, error) {
	remote.Start()
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("error generating key: %s", err)
	}
	p2pHost, err := p2p.NewLibP2PHost(ctx, key, 0)
	if err != nil {
		return nil, fmt.Errorf("error setting up p2p host: %s", err)
	}
	if _, err = p2pHost.Bootstrap(p2p.BootstrapNodes()); err != nil {
		return nil, err
	}
	if err = p2pHost.WaitForBootstrap(len(group.Signers), 15*time.Second); err != nil {
		return nil, err
	}

	group.SetupAllRemoteActors(&key.PublicKey)
	return p2pHost, nil
}

func TestCreateChainTree(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := SetupTupeloNotaryGroup(ctx, true)
	require.Nil(t, err)

	node, err := setupRemote(ctx, group)
	require.Nil(t, err)

	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	ps := remote.NewNetworkPubSub(node.GetPubSub())

	tup := &Tupelo{
		Store:        nodestore.MustMemoryStore(ctx),
		NotaryGroup:  group,
		PubSubSystem: ps,
	}

	_, err = tup.CreateChainTree(key)
	require.Nil(t, err)
}

func TestGetTip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := SetupTupeloNotaryGroup(ctx, true)
	require.Nil(t, err)

	node, err := setupRemote(ctx, group)
	require.Nil(t, err)

	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	ps := remote.NewNetworkPubSub(node.GetPubSub())

	tup := &Tupelo{
		Store:        nodestore.MustMemoryStore(ctx),
		NotaryGroup:  group,
		PubSubSystem: ps,
	}

	tree, err := tup.CreateChainTree(key)
	require.Nil(t, err)

	time.Sleep(100 * time.Millisecond)
	tip, err := tup.GetTip(tree.MustId())
	require.Nil(t, err)

	require.Equal(t, tree.Tip(), tip)
}
