package summer

import (
	"context"
	"path/filepath"

	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"

	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/courts/artifact"
	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/game"
	handlers "github.com/quorumcontrol/jasons-game/handlers"
	inventoryHandlers "github.com/quorumcontrol/jasons-game/handlers/inventory"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/service"
)

var log = logging.Logger("summer")

const prizeHandlerTreeName = "winning-prize-handler"
const artifactPickupHandlerTreeName = "artifact-pickup-handler"

type summerConfig struct {
	Spawning []struct {
		Locations []string
		Forgers   []string
	}
}

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

func (c *SummerCourt) ids() (map[string]interface{}, error) {
	return c.court.Ids()
}

func (c *SummerCourt) config() *summerConfig {
	ids, err := c.ids()
	if err != nil {
		panic(errors.Wrap(err, "error fetching court ids"))
	}

	cfg := &summerConfig{}
	err = config.ReadYaml(filepath.Join(c.configPath, "summer/config.yml"), cfg, ids)
	if err != nil {
		panic(errors.Wrap(err, "error fetching config"))
	}

	return cfg
}

func (c *SummerCourt) setupArtifactPickupAndSpawn(actorCtx actor.Context, config *summerConfig) {
	handlerTree, err := court.FindOrCreateNamedTree(c.net, artifactPickupHandlerTreeName)
	if err != nil {
		panic(err)
	}
	handler := inventoryHandlers.NewUnrestrictedRemoveHandler(c.net)

	servicePID, err := actorCtx.SpawnNamed(service.NewServiceActorPropsWithTree(c.net, handler, handlerTree), artifactPickupHandlerTreeName)
	if err != nil {
		panic(err)
	}
	// This is the same as the handlerTree.MustId(), but just ensures it has started up
	handlerDid, err := actorCtx.RequestFuture(servicePID, &service.GetServiceDid{}, 30*time.Second).Result()
	if err != nil {
		panic(err)
	}
	log.Infof("%s handler started with did %s", artifactPickupHandlerTreeName, handlerDid)

	for _, spawnConfig := range config.Spawning {
		for _, spawnLocation := range spawnConfig.Locations {
			locationHandler, err := handlers.FindHandlerForTree(c.net, spawnLocation)
			if err != nil {
				panic(errors.Wrap(err, "getting location handler"))
			}
			if locationHandler == nil || locationHandler.Did() != handlerDid {
				locTree, err := c.net.GetTree(spawnLocation)
				if err != nil {
					panic(errors.Wrap(err, "getting loc tree"))
				}
				loc := game.NewLocationTree(c.net, locTree)
				err = loc.SetHandler(handlerDid.(string))
				if err != nil {
					panic(errors.Wrap(err, "getting loc tree"))
				}
			}
		}

		respawner, err := artifact.NewRespawnActor(c.ctx, &artifact.RespawnActorConfig{
			Network:    c.net,
			Locations:  spawnConfig.Locations,
			Forgers:    spawnConfig.Forgers,
			ConfigPath: c.configPath,
		})
		if err != nil {
			panic(err)
		}
		respawner.Start(actorCtx)
	}
}

func (c *SummerCourt) setupWinningPrizeHandler(actorCtx actor.Context, config *summerConfig) {
	handler, err := court.NewPrizeHandler(&court.PrizeHandlerConfig{
		Court:           c.court,
		PrizeConfigPath: filepath.Join(c.configPath, "summer/prize_config.yml"),
	})
	if err != nil {
		panic(err)
	}
	servicePID, err := actorCtx.SpawnNamed(service.NewServiceActorPropsWithTree(c.net, handler, handler.Tree()), prizeHandlerTreeName)
	if err != nil {
		panic(err)
	}
	// This is the same as the handlerTree.MustId(), but just ensures it has started up
	handlerDid, err := actorCtx.RequestFuture(servicePID, &service.GetServiceDid{}, 30*time.Second).Result()
	if err != nil {
		panic(err)
	}
	log.Infof("%s handler started with did %s", prizeHandlerTreeName, handlerDid)
}

func (c *SummerCourt) initialize(actorCtx actor.Context) {
	err := c.court.Import(c.configPath)
	if err != nil {
		panic(err)
	}

	config := c.config()
	c.setupArtifactPickupAndSpawn(actorCtx, config)
	c.setupWinningPrizeHandler(actorCtx, config)
}

func (c *SummerCourt) Receive(actorCtx actor.Context) {
	switch actorCtx.Message().(type) {
	case *actor.Started:
		c.initialize(actorCtx)
	}
}
