// +build integration

package game

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
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
	jsonBytes, err := ioutil.ReadFile(path.Join(path.Dir(filename), "../devdocker/localkeys/public-keys.json"))
	if err != nil {
		return nil, err
	}
	var keySet []*publicKeySet
	if err := json.Unmarshal(jsonBytes, &keySet); err != nil {
		return nil, err
	}

	return keySet, nil
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

func TestFullIntegration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	path := "/tmp/test-full-game"

	err = os.RemoveAll(path)
	require.Nil(t, err)

	err = os.MkdirAll(path, 0755)
	require.Nil(t, err)

	defer os.RemoveAll(path)

	net, err := network.NewRemoteNetwork(ctx, group, path)
	require.Nil(t, err)

	rootCtx := actor.EmptyRootContext

	stream := ui.NewTestStream()

	uiActor, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), "test-integration-ui")
	require.Nil(t, err)
	defer rootCtx.Stop(uiActor)

	playerTree, err := GetOrCreatePlayerTree(net)
	require.Nil(t, err)
	gameActor, err := rootCtx.SpawnNamed(NewGameProps(playerTree, uiActor, net),
		"test-integration-game")
	require.Nil(t, err)
	defer rootCtx.Stop(gameActor)

	readyFut := rootCtx.RequestFuture(gameActor, &ping{}, 15*time.Second)
	// wait on the game actor being ready
	_, err = readyFut.Result()
	require.Nil(t, err)

	someTree, err := net.CreateChainTree()
	require.Nil(t, err)

	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: fmt.Sprintf("connect location %s as enter dungeon", someTree.MustId())})
	time.Sleep(100 * time.Millisecond)
	msgs := filterUserMessages(t, stream.GetMessages())
	require.Len(t, msgs, 2)

	rootCtx.Send(gameActor, &jasonsgame.UserInput{Message: "enter dungeon"})
	time.Sleep(100 * time.Millisecond)
	msgs = filterUserMessages(t, stream.GetMessages())
	require.Len(t, msgs, 3)
}
