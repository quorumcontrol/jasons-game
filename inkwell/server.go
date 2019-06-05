package inkwell

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"

	"github.com/quorumcontrol/jasons-game/inkwell/ink"
	"github.com/quorumcontrol/jasons-game/network"
)

type Inkwell struct {
	group     *types.NotaryGroup
	statePath string
	net       network.Network
	inkSource ink.Source
}

type InkwellConfig struct {
	Localnet  bool
	StatePath string
}

func NewServer(ctx context.Context, cfg InkwellConfig) (*Inkwell, error) {
	group, err := network.SetupTupeloNotaryGroup(ctx, cfg.Localnet)
	if err != nil {
		return nil, errors.Wrap(err,"error setting up notary group")
	}

	net, err := network.NewRemoteNetwork(ctx, group, cfg.StatePath)

	inkDID := os.Getenv("INK_DID")

	if inkDID == "" {
		fmt.Println("Generating random ink source chaintree")
	}

	sourceCfg := ink.ChainTreeInkSourceConfig{
		Net: net,
	}

	inkSource, err := ink.NewChainTreeInkSource(sourceCfg)
	if err != nil {
		panic(errors.Wrap(err, "error getting ink source"))
	}

	return &Inkwell{
		group:     group,
		statePath: cfg.StatePath,
		net:       net,
		inkSource: inkSource,
	}, nil
}
