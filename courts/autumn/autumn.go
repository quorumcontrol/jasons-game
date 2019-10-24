package autumn

import (
	"context"
	"path/filepath"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"

	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/service"
)

var log = logging.Logger("autumn")

const elementCombinerConcurrency = 200

type AutumnConfig struct {
	PrizeFailMsg   string                `yaml:"prize_fail_msg"`
	WinningElement int                   `yaml:"winning_element"`
	Elements       []*element            `yaml:"elements"`
	Binder         []*elementCombination `yaml:"binder"`
	Weaver         []*elementCombination `yaml:"weaver"`
}

type AutumnCourt struct {
	ctx        context.Context
	net        network.Network
	court      *court.Court
	config     *AutumnConfig
	configPath string
}

func New(ctx context.Context, net network.Network, configPath string) *AutumnCourt {
	return &AutumnCourt{
		ctx:        ctx,
		net:        net,
		court:      court.New(ctx, net, "autumn"),
		configPath: configPath,
	}
}

func (c *AutumnCourt) Start() {
	court.SpawnCourt(c.ctx, c)
}

func (c *AutumnCourt) Ids() (map[string]interface{}, error) {
	return c.court.Ids()
}

func (c *AutumnCourt) parseConfig() *AutumnConfig {
	log.Info("parsing config")
	ids, err := c.Ids()
	if err != nil {
		panic(errors.Wrap(err, "error fetching court Ids"))
	}

	log.Debugf("IDs: %+v", ids)

	cfg := &AutumnConfig{}
	err = config.ReadYaml(filepath.Join(c.configPath, "autumn/config.yml"), cfg, ids)
	if err != nil {
		panic(errors.Wrap(err, "error fetching config"))
	}

	if cfg.PrizeFailMsg == "" {
		cfg.PrizeFailMsg = "you have failed"
	}

	if cfg.WinningElement == 0 {
		panic("must set winning_element in autumn/config.yml")
	}

	return cfg
}

func (c *AutumnCourt) spawnPrizeHandler(actorCtx actor.Context) {
	handler, err := NewAutumnPrizeHandler(c)
	if err != nil {
		panic(errors.Wrap(err, "creating prize handler"))
	}

	_, err = c.court.SpawnHandler(actorCtx, handler)
	if err != nil {
		panic(err)
	}
}

func (c *AutumnCourt) setupArtifactHandler(actorCtx actor.Context) {
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

func (c *AutumnCourt) setupCombinationHandler(actorCtx actor.Context, name string, elements []*element, combinations []*elementCombination) {
	locationDidUncast, err := c.court.Resolve([]string{"tree", "data", "ids", "Locations", name})
	if err != nil {
		panic(err)
	}
	locationDid, ok := locationDidUncast.(string)
	if !ok {
		panic("Could not find location for " + name)
	}

	handler, err := NewElementCombinerHandler(&ElementCombinerHandlerConfig{
		Name:         name,
		Network:      c.net,
		Location:     locationDid,
		Elements:     elements,
		Combinations: combinations,
	})
	if err != nil {
		panic(err)
	}

	_, err = actorCtx.SpawnNamed(service.NewServiceActorPropsWithConfig(&service.ServiceActorConfig{
		Network:     c.net,
		Handler:     handler,
		Tree:        handler.Tree(),
		Concurrency: elementCombinerConcurrency,
	}), handler.Name())

	log.Infof("%s handler started with did %s", handler.Name(), handler.Tree().MustId())

	if err != nil {
		panic(err)
	}
}

func (c *AutumnCourt) initialize(actorCtx actor.Context) {
	err := c.court.Import(c.configPath)
	if err != nil {
		log.Errorf("error importing config from %s: %v", c.configPath, err)
		panic(err)
	}

	c.config = c.parseConfig()

	c.setupCombinationHandler(actorCtx, "weaver", c.config.Elements, c.config.Weaver)
	c.setupCombinationHandler(actorCtx, "binder", c.config.Elements, c.config.Binder)
	c.setupArtifactHandler(actorCtx)
	c.spawnPrizeHandler(actorCtx)
}

func (c *AutumnCourt) Receive(actorCtx actor.Context) {
	switch actorCtx.Message().(type) {
	case *actor.Started:
		log.Info("initializing autumn court actor")
		c.initialize(actorCtx)
	}
}
