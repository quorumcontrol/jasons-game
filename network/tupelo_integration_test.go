// +build integration

package network

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-client/bls"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/storage"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
	"github.com/stretchr/testify/require"
)

type publicKeySet struct {
	BlsHexPublicKey   string `json:"blsHexPublicKey,omitempty"`
	EcdsaHexPublicKey string `json:"ecdsaHexPublicKey,omitempty"`
	PeerIDBase58Key   string `json:"peerIDBase58Key,omitempty"`
}

func loadSignerKeys() ([]*publicKeySet, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("No caller information")
	}
	jsonBytes, err := ioutil.ReadFile(path.Join(path.Dir(filename), "test-signer-keys/public-keys.json"))
	if err != nil {
		return nil, err
	}
	var keySet []*publicKeySet
	if err := json.Unmarshal(jsonBytes, &keySet); err != nil {
		return nil, err
	}

	return keySet, nil
}

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

	remote.NewRouter(p2pHost)
	group.SetupAllRemoteActors(&key.PublicKey)
	return p2pHost, nil
}

func setupNotaryGroup(ctx context.Context) (*types.NotaryGroup, error) {
	keys, err := loadSignerKeys()
	if err != nil {
		return nil, err
	}
	group := types.NewNotaryGroup("hardcodedprivatekeysareunsafe")
	for _, keySet := range keys {
		ecdsaBytes := hexutil.MustDecode(keySet.EcdsaHexPublicKey)
		verKeyBytes := hexutil.MustDecode(keySet.BlsHexPublicKey)
		ecdsaPubKey, err := crypto.UnmarshalPubkey(ecdsaBytes)
		if err != nil {
			return nil, err
		}
		signer := types.NewRemoteSigner(ecdsaPubKey, bls.BytesToVerKey(verKeyBytes))
		group.AddSigner(signer)
	}

	return group, nil
}

func TestCreateChainTree(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	node, err := setupRemote(ctx, group)
	require.Nil(t, err)

	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	ps := remote.NewNetworkPubSub(node)
	
	tup := &Tupelo{
		key:   key,
		Store: nodestore.NewStorageBasedStore(storage.NewMemStorage()),
		NotaryGroup: group,
		PubSubSystem: ps,
	}

	_,err = tup.CreateChainTree()
	require.Nil(t,err)
}

func TestGetTip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	node, err := setupRemote(ctx, group)
	require.Nil(t, err)

	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	ps := remote.NewNetworkPubSub(node)
	
	tup := &Tupelo{
		key:   key,
		Store: nodestore.NewStorageBasedStore(storage.NewMemStorage()),
		NotaryGroup: group,
		PubSubSystem: ps,
	}

	tree,err := tup.CreateChainTree()
	require.Nil(t,err)

	time.Sleep(100 * time.Millisecond)
	tip,err := tup.GetTip(tree.MustId())
	require.Nil(t,err)

	require.Equal(t, tree.Tip(), tip)
}
