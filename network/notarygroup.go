package network

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	packr "github.com/gobuffalo/packr/v2"
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
	localBox := packr.New("localKeys", "../devdocker/localkeys")
	testnetBox := packr.New("testnetKeys", "../devdocker/testnetkeys")

	var jsonBytes []byte
	var err error

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

func SetupTupeloNotaryGroup(ctx context.Context, connectToLocalnet bool) (*types.NotaryGroup, error) {
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
