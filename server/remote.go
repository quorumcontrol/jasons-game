package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
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
