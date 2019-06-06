package server

import (
	"context"
	"os"
	"strings"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/inkwell/ink"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("inkwell")

type Inkwell struct {
	group     *types.NotaryGroup
	dataStore datastore.Batching
	net       network.Network
	inkSource ink.Source
	tokenName string
}

type InkwellConfig struct {
	Local    bool
	S3Region string
	S3Bucket string
}

func NewServer(ctx context.Context, cfg InkwellConfig) (*Inkwell, error) {
	group, err := network.SetupTupeloNotaryGroup(ctx, cfg.Local)
	if err != nil {
		return nil, errors.Wrap(err,"error setting up notary group")
	}

	ds, err := config.S3DataStore(cfg.Local, cfg.S3Region, cfg.S3Bucket)
	if err != nil {
		panic(errors.Wrap(err, "error getting S3 data store"))
	}

	net, err := network.NewRemoteNetwork(ctx, group, ds)
	if err != nil {
		panic(errors.Wrap(err, "error setting up remote network"))
	}

	inkDID := os.Getenv("INK_DID")

	if inkDID == "" {
		panic("INK_DID must be set")
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
		dataStore: ds,
		net:       net,
		inkSource: inkSource,
		tokenName: strings.Join([]string{inkDID, "ink"}, ":"),
	}, nil
}
