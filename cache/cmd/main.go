package main

import (
	"context"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/jasons-game/cache"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
)

func main() {
	config.MustSetLogLevel("jgcache", "info")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, false)
	if err != nil {
		panic(err)
	}

	signingKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	netCfg := &network.RemoteNetworkConfig{
		NotaryGroup:   notaryGroup,
		KeyValueStore: config.MemoryDataStore(),
		SigningKey:    signingKey,
	}

	net, err := network.NewRemoteNetworkWithConfig(ctx, netCfg)
	if err != nil {
		panic(err)
	}

	err = cache.Export(net)
	if err != nil {
		panic(err)
	}
}
