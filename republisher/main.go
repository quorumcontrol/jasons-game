package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/quorumcontrol/jasons-game/network"

	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gobuffalo/packr/v2"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/shibukawa/configdir"
)

const sessionStorageDir = "session-storage"

var log = logging.Logger("gamenetwork")

func doIt(ctx context.Context) error {
	err := logging.SetLogLevel("gamenetwork", "info")
	if err != nil {
		return errors.Wrap(err, "error setting log level")
	}

	log.Infof("Starting republisher")

	group, err := setupNotaryGroup(ctx, false)
	if err != nil {
		panic(errors.Wrap(err, "setting up notary group"))
	}

	configDirs := configdir.New("tupelo", "jasons-game")
	folders := configDirs.QueryFolders(configdir.Global)
	folder := configDirs.QueryFolderContainsFile(sessionStorageDir)
	if folder == nil {
		if err := folders[0].CreateParentDir(sessionStorageDir); err != nil {
			panic(err)
		}
	}

	sessionPath := filepath.Join(folders[0].Path, sessionStorageDir)

	statePath := filepath.Join(sessionPath, filepath.Base("12345"))
	if err := os.MkdirAll(statePath, 0750); err != nil {
		panic(errors.Wrap(err, "error creating session storage"))
	}
	net, err := network.NewRemoteNetwork(ctx, group, statePath)
	if err != nil {
		panic(errors.Wrap(err, "setting up network"))
	}

	err = net.(*network.RemoteNetwork).RepublishAll()
	if err != nil {
		panic(errors.Wrap(err, "error on publish"))
	}
	time.Sleep(5 * time.Second)
	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := doIt(ctx)
	if err != nil {
		panic(err)
	}
}

type publicKeySet struct {
	BlsHexPublicKey   string `json:"blsHexPublicKey,omitempty"`
	EcdsaHexPublicKey string `json:"ecdsaHexPublicKey,omitempty"`
	PeerIDBase58Key   string `json:"peerIDBase58Key,omitempty"`
}

func loadSignerKeys(connectToLocalnet bool) ([]*publicKeySet, error) {
	localBox := packr.New("localKeys", "../devdocker/localkeys")
	testnetBox := packr.New("testnetKeys", "../devdocker/testnetkeys")

	var jsonBytes []byte
	var err error

	// TODO: Referencing devdocker dir here seems gross; should maybe rethink this
	if connectToLocalnet {
		jsonBytes, err = localBox.Find("public-keys.json")
	} else {
		jsonBytes, err = testnetBox.Find("public-keys.json")
	}

	if err != nil {
		return nil, err
	}
	var keySet []*publicKeySet
	if err := json.Unmarshal(jsonBytes, &keySet); err != nil {
		return nil, err
	}

	return keySet, nil
}

func setupNotaryGroup(ctx context.Context, connectToLocalnet bool) (*types.NotaryGroup, error) {
	keys, err := loadSignerKeys(connectToLocalnet)
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
