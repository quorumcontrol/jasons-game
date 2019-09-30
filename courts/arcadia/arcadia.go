package arcadia

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/utils/stringslice"
)

var log = logging.Logger("arcadia")

type ArcadiaCourt struct {
	ctx        context.Context
	net        network.Network
	court      *court.Court
	configPath string
	altarDids  []string
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

func (c *ArcadiaCourt) setupArtifactSpawnHandler(actorCtx actor.Context) {
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
}

var altars = []string{"altarone", "altartwo", "altarthree", "altarfour", "altarfive"}

func (c *ArcadiaCourt) fetchAltarDids() ([]string, error) {
	courtIds, err := c.court.Ids()
	altarDids := make([]string, len(altars))

	if err != nil {
		return altarDids, err
	}

	uncastLocationIds, ok := courtIds["Locations"].(map[string]interface{})
	if !ok {
		return altarDids, err
	}

	for locationName, locationDid := range uncastLocationIds {
		altarIndex := stringslice.Index(altars, locationName)
		if altarIndex >= 0 {
			locationDidStr, ok := locationDid.(string)
			if !ok {
				return altarDids, err
			}
			altarDids[altarIndex] = locationDidStr
		}
	}

	return altarDids, err
}

func (c *ArcadiaCourt) configureAltars() {
	for _, altarDid := range c.altarDids {
		if altarDid == "" {
			panic("altar does not exist")
		}

		altarTree, err := c.net.GetTree(altarDid)
		if err != nil {
			panic(errors.Wrap(err, "error finding altar "+altarDid))
		}

		_, err = c.net.UpdateChainTree(altarTree, "jasons-game/use-per-player-inventory", true)
		if err != nil {
			panic(errors.Wrap(err, "error updating altar "+altarDid))
		}
	}
}

func (c *ArcadiaCourt) setupPrizeHandler(actorCtx actor.Context) {
	handler, err := NewEndGamePrizeHandler(c)
	if err != nil {
		panic(err)
	}

	_, err = c.court.SpawnHandler(actorCtx, handler)
	if err != nil {
		panic(err)
	}
}

func (c *ArcadiaCourt) initialize(actorCtx actor.Context) {
	err := c.court.Import(c.configPath)
	if err != nil {
		panic(err)
	}

	c.altarDids, err = c.fetchAltarDids()
	if err != nil {
		panic(err)
	}

	c.configureAltars()
	c.setupArtifactSpawnHandler(actorCtx)
	c.setupPrizeHandler(actorCtx)
}

func (c *ArcadiaCourt) Receive(actorCtx actor.Context) {
	switch actorCtx.Message().(type) {
	case *actor.Started:
		c.initialize(actorCtx)
	}
}
