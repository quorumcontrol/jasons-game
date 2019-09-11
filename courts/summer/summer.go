package summer

import (
	"context"
	"path/filepath"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"

	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("summer")

type SummerCourt struct {
	ctx        context.Context
	net        network.Network
	court      *court.Court
	configPath string
}

func New(ctx context.Context, net network.Network, configPath string) *SummerCourt {
	return &SummerCourt{
		ctx:        ctx,
		net:        net,
		court:      court.New(ctx, net, "summer"),
		configPath: configPath,
	}
}

func (c *SummerCourt) Start() {
	court.SpawnCourt(c.ctx, c)
}

func (c *SummerCourt) setupArtifactHandler(actorCtx actor.Context) {
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
	log.Infof("%s handler started with did %s", handler.Name(), handler.Tree().MustId())
}

func (c *SummerCourt) setupWinningPrizeHandler(actorCtx actor.Context) {
	handler, err := court.NewPrizeHandler(&court.PrizeHandlerConfig{
		Court:           c.court,
		PrizeConfigPath: filepath.Join(c.configPath, "summer/prize_config.yml"),
	})
	if err != nil {
		panic(err)
	}
	_, err = c.court.SpawnHandler(actorCtx, handler)
	if err != nil {
		panic(err)
	}
	log.Infof("%s handler started with did %s", handler.Name(), handler.Tree().MustId())
}

func (c *SummerCourt) initialize(actorCtx actor.Context) {
	err := c.court.Import(c.configPath)
	if err != nil {
		panic(err)
	}

	c.setupArtifactHandler(actorCtx)
	c.setupWinningPrizeHandler(actorCtx)
}

func (c *SummerCourt) Receive(actorCtx actor.Context) {
	switch actorCtx.Message().(type) {
	case *actor.Started:
		c.initialize(actorCtx)
	}
}
