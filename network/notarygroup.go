package network

import (
	"context"
	"fmt"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
)

func loadSignerConfig(connectToLocalnet bool) (*types.Config, error) {
	// TODO: Referencing devdocker dir here seems gross; should maybe rethink this
	localBox := packr.New("localKeys", "../devdocker/localkeys")
	testnetBox := packr.New("testnetKeys", "../devdocker/testnetkeys")

	var tomlBytes []byte
	var err error

	if connectToLocalnet {
		tomlBytes, err = localBox.Find("notarygroup.toml")
	} else {
		tomlBytes, err = testnetBox.Find("notarygroup.toml")
	}

	ngConfig, err := types.TomlToConfig(string(tomlBytes))
	if err != nil {
		return nil, fmt.Errorf("error loading notary group config")
	}

	return ngConfig, nil
}

func SetupTupeloNotaryGroup(ctx context.Context, connectToLocalnet bool) (*types.NotaryGroup, error) {
	config, err := loadSignerConfig(connectToLocalnet)
	if err != nil {
		return nil, err
	}

	group := types.NewNotaryGroupFromConfig(config)

	for _, keySet := range config.Signers {
		signer := types.NewRemoteSigner(keySet.DestKey, keySet.VerKey)
		group.AddSigner(signer)
	}

	return group, nil
}
