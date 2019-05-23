package server

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gobuffalo/packr/v2"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
)

type publicKeySet struct {
	BlsHexPublicKey   string `json:"blsHexPublicKey,omitempty"`
	EcdsaHexPublicKey string `json:"ecdsaHexPublicKey,omitempty"`
	PeerIDBase58Key   string `json:"peerIDBase58Key,omitempty"`
}

func loadSignerKeys(connectToLocalnet bool) ([]*publicKeySet, error) {
	// TODO: Referencing devdocker dir here seems gross; should maybe rethink this
	box := packr.New("keys", "../devdocker")

	var keyDir string
	if connectToLocalnet {
		keyDir = "localkeys"
	} else {
		keyDir = "testnetkeys"
	}

	jsonBytes, err := box.Find(filepath.Join(keyDir, "public-keys.json"))
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
