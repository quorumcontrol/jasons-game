package basic

import (
	"context"
	"path/filepath"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"

	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/network"
)

type BasicCourt struct {
	ctx        context.Context
	net        network.Network
	court      *court.Court
	configPath string
	log        logging.StandardLogger
}

func New(ctx context.Context, net network.Network, configPath string, name string) *BasicCourt {
	return &BasicCourt{
		ctx:        ctx,
		net:        net,
		court:      court.New(ctx, net, name),
		configPath: configPath,
		log:        logging.Logger(name),
	}
}

func (c *BasicCourt) Start() {
	court.SpawnCourt(c.ctx, c)
}

func (c *BasicCourt) setupArtifactHandler(actorCtx actor.Context) {
	handler, err := court.NewArtifactSpawnHandler(&court.ArtifactSpawnHandlerConfig{
		Court:      c.court,
		ConfigPath: c.configPath,
	})
	if err != nil {
		panic(err)
	}
	_, err = c.court.SpawnHandler(actorCtx, handler)
	if err != nil {
		panic(err)
	}
	c.log.Infof("%s handler started with did %s", handler.Name(), handler.Tree().MustId())
}

func (c *BasicCourt) setupWinningPrizeHandler(actorCtx actor.Context) {
	handler, err := court.NewPrizeHandler(&court.PrizeHandlerConfig{
		Court:           c.court,
		PrizeConfigPath: filepath.Join(c.configPath, c.court.Name(), "prize_config.yml"),
	})
	if err != nil {
		panic(err)
	}
	_, err = c.court.SpawnHandler(actorCtx, handler)
	if err != nil {
		panic(err)
	}
	c.log.Infof("%s handler started with did %s", handler.Name(), handler.Tree().MustId())
}

func (c *BasicCourt) initialize(actorCtx actor.Context) {
	err := c.court.Import(c.configPath)
	if err != nil {
		panic(err)
	}

	c.setupArtifactHandler(actorCtx)
	c.setupWinningPrizeHandler(actorCtx)
}

func (c *BasicCourt) Receive(actorCtx actor.Context) {
	switch actorCtx.Message().(type) {
	case *actor.Started:
		c.initialize(actorCtx)
	}
}
