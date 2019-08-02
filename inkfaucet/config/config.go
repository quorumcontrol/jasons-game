package config

import (
	"context"

	"github.com/ipfs/go-datastore"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
)

type InkFaucetConfig struct {
	Local       bool
	S3Region    string
	S3Bucket    string
	InkOwnerDID string
}

type InkFaucet struct {
	NotaryGroup *types.NotaryGroup
	DataStore   datastore.Batching
	Net         network.Network
}

func Setup(ctx context.Context, cfg InkFaucetConfig) (*InkFaucet, error) {
	iw := InkFaucet{}

	group, err := network.SetupTupeloNotaryGroup(ctx, cfg.Local)
	if err != nil {
		return nil, errors.Wrap(err, "error setting up notary group")
	}
	iw.NotaryGroup = group

	ds, err := config.S3DataStore(cfg.Local, cfg.S3Region, cfg.S3Bucket)
	if err != nil {
		return nil, errors.Wrap(err, "error getting S3 data store")
	}
	iw.DataStore = ds

	signingKey, err := network.GetOrCreateStoredPrivateKey(ds)
	if err != nil {
		return nil, errors.Wrap(err, "error getting private signingKey")
	}

	netCfg := &network.RemoteNetworkConfig{
		NotaryGroup:   group,
		KeyValueStore: ds,
		SigningKey:    signingKey,
	}

	net, err := network.NewRemoteNetworkWithConfig(ctx, netCfg)
	if err != nil {
		return nil, errors.Wrap(err, "error setting up remote network")
	}
	iw.Net = net.(*network.RemoteNetwork)

	return &iw, nil
}
