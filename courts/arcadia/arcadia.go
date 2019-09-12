package arcadia

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"

	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("arcadia")

type ArcadiaCourt struct {
	ctx        context.Context
	net        network.Network
	court      *court.Court
	configPath string
}

func New(ctx context.Context, net network.Network, configPath string) *ArcadiaCourt {
	return &ArcadiaCourt{
		ctx:        ctx,
		net:        net,
		court:      court.New(ctx, net, "arcadia"),
		configPath: configPath,
	}
}

func (c *ArcadiaCourt) Start() {
	court.SpawnCourt(c.ctx, c)
}

func (c *ArcadiaCourt) setupArtifactHandler(actorCtx actor.Context) {
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

func (c *ArcadiaCourt) initialize(actorCtx actor.Context) {
	err := c.court.Import(c.configPath)
	if err != nil {
		panic(err)
	}

	c.setupArtifactHandler(actorCtx)
}

func (c *ArcadiaCourt) Receive(actorCtx actor.Context) {
	switch actorCtx.Message().(type) {
	case *actor.Started:
		c.initialize(actorCtx)
	}
}
