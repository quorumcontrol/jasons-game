package config

import (
	"context"
	"crypto/ecdsa"

	"github.com/ipfs/go-datastore"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
)

type InkFaucetConfig struct {
	Local        bool
	InkOwnerDID  string
	InkFaucetDID string
	PrivateKey   *ecdsa.PrivateKey
}

type InkFaucet struct {
	NotaryGroup *types.NotaryGroup
	DataStore   datastore.Batching
	Net         network.Network
}

func Setup(ctx context.Context, cfg InkFaucetConfig) (*InkFaucet, error) {
	inkFaucet := InkFaucet{}

	group, err := network.SetupTupeloNotaryGroup(ctx, cfg.Local)
	if err != nil {
		return nil, errors.Wrap(err, "error setting up notary group")
	}
	inkFaucet.NotaryGroup = group

	ds := config.MemoryDataStore()

	inkFaucet.DataStore = ds

	netCfg := &network.RemoteNetworkConfig{
		NotaryGroup:   group,
		KeyValueStore: ds,
		SigningKey:    cfg.PrivateKey,
	}

	net, err := network.NewRemoteNetworkWithConfig(ctx, netCfg)
	if err != nil {
		return nil, errors.Wrap(err, "error setting up remote network")
	}
	inkFaucet.Net = net

	return &inkFaucet, nil
}
