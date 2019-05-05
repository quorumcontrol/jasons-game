package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	deeplogging "github.com/whyrusleeping/go-logging"
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
	jsonBytes, err := ioutil.ReadFile(path.Join(path.Dir(filename), "network/test-signer-keys/public-keys.json"))
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
	group := types.NewNotaryGroup("tupelo-in-local-docker")
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

func main() {
	os.Mkdir("log", 0755)
	f, _ := os.Create(filepath.Join("log", fmt.Sprintf("jasons-%d.log", time.Now().Unix())))
	lgbe := deeplogging.NewLogBackend(f, "", 0)
	deeplogging.SetBackend(lgbe)

	err := logging.SetLogLevel("*", "INFO")
	if err != nil {
		panic(errors.Wrap(err, "error setting loglevel"))
	}
	err = logging.SetLogLevel("ui", "DEBUG")
	if err != nil {
		panic(errors.Wrap(err, "error setting loglevel"))
	}
	err = logging.SetLogLevel("game", "DEBUG")
	if err != nil {
		panic(errors.Wrap(err, "error setting loglevel"))
	}
	err = logging.SetLogLevel("swarm2", "error")
	if err != nil {
		panic(errors.Wrap(err, "error setting loglevel"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, err := setupNotaryGroup(ctx)
	if err != nil {
		panic(errors.Wrap(err, "setting up notary group"))
	}

	path := "/tmp/localplay"

	os.RemoveAll(path)
	os.MkdirAll(path, 0755)
	defer os.RemoveAll(path)

	net, err := network.NewRemoteNetwork(ctx, group, path)
	if err != nil {
		panic(errors.Wrap(err, "setting up notary group"))
	}

	rootCtx := actor.EmptyRootContext
	ui, err := rootCtx.SpawnNamed(ui.NewUIProps(), "ui")
	if err != nil {
		panic(fmt.Errorf("error running UI: %v", err))
	}

	gameActor, err := rootCtx.SpawnNamed(game.NewGameProps(ui, net), "game")
	if err != nil {
		panic(fmt.Errorf("error running UI: %v", err))
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()
	fmt.Println("hit ctrl-C one more time to exit")
	<-done
	gameActor.Stop()
}
