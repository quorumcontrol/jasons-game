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
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/tupelo-go-client/bls"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
	"github.com/stretchr/testify/assert"
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
	jsonBytes, err := ioutil.ReadFile(path.Join(path.Dir(filename), "../network/test-signer-keys/public-keys.json"))
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

	os.RemoveAll(path)
	os.MkdirAll(path, 0755)
	defer os.RemoveAll(path)

	net, err := network.NewRemoteNetwork(ctx, group, path)
	require.Nil(t, err)

	rootCtx := actor.EmptyRootContext

	simulatedUI, err := rootCtx.SpawnNamed(ui.NewSimulatedUIProps(), "ui")
	require.Nil(t, err)
	defer simulatedUI.Stop()

	gameActor, err := rootCtx.SpawnNamed(NewGameProps(simulatedUI, net), "game")
	require.Nil(t, err)
	defer gameActor.Stop()

	readyFut := rootCtx.RequestFuture(gameActor, &ping{}, 15 * time.Second)
	// wait on the game actor being ready
	_,err = readyFut.Result()
	require.Nil(t,err)

	rootCtx.Send(gameActor, &ui.UserInput{Message: "north"})

	time.Sleep(200 * time.Millisecond)

	fut := rootCtx.RequestFuture(simulatedUI, &ui.GetEventsFromSimulator{}, 40*time.Second)
	evts, err := fut.Result()
	require.Nil(t, err)

	require.Len(t, evts.([]interface{}), 3)
	assert.IsType(t, &navigator.Location{}, evts.([]interface{})[1])
	assert.IsType(t, &navigator.Location{}, evts.([]interface{})[2])
}
