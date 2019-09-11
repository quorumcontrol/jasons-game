package autumn

import (
	"context"
	"path/filepath"

	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"

	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/service"
)

var log = logging.Logger("autumn")

type autumnConfig struct {
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
	config     *autumnConfig
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

func (c *AutumnCourt) ids() (map[string]interface{}, error) {
	return c.court.Ids()
}

func (c *AutumnCourt) parseConfig() *autumnConfig {
	ids, err := c.ids()
	if err != nil {
		panic(errors.Wrap(err, "error fetching court ids"))
	}

	cfg := &autumnConfig{}
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

func (c *AutumnCourt) spawnPrizeHandler(actorCtx actor.Context) error {
	handler, err := NewAutumnPrizeHandler(c)
	if err != nil {
		return errors.Wrap(err, "creating prize handler")
	}

	servicePID, err := actorCtx.SpawnNamed(service.NewServiceActorPropsWithTree(c.net, handler, handler.Tree()), "autumn-prize-handler")
	if err != nil {
		return err
	}
	handlerDid, err := actorCtx.RequestFuture(servicePID, &service.GetServiceDid{}, 30*time.Second).Result()
	if err != nil {
		return err
	}
	log.Errorf("autumn prizehandler started with did %s", handlerDid)

	return nil
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

	handlerName := name + "-handler"
	handlerTree, err := court.FindOrCreateNamedTree(c.net, handlerName)
	if err != nil {
		panic(err)
	}

	handler, err := NewElementCombinerHandler(&ElementCombinerHandlerConfig{
		Did:          handlerTree.MustId(),
		Network:      c.net,
		Location:     locationDid,
		Elements:     elements,
		Combinations: combinations,
	})
	if err != nil {
		panic(err)
	}
	servicePID, err := actorCtx.SpawnNamed(service.NewServiceActorPropsWithTree(c.net, handler, handlerTree), handlerName)
	if err != nil {
		panic(err)
	}
	// This is the same as the handlerTree.MustId(), but just ensures it has started up
	handlerDid, err := actorCtx.RequestFuture(servicePID, &service.GetServiceDid{}, 30*time.Second).Result()
	if err != nil {
		panic(err)
	}
	log.Infof("%s handler started with did %s", handlerName, handlerDid)
}

func (c *AutumnCourt) initialize(actorCtx actor.Context) {
	err := c.court.Import(c.configPath)
	if err != nil {
		panic(err)
	}

	c.config = c.parseConfig()

	c.setupCombinationHandler(actorCtx, "weaver", c.config.Elements, c.config.Weaver)
	c.setupCombinationHandler(actorCtx, "binder", c.config.Elements, c.config.Binder)

	err = c.spawnPrizeHandler(actorCtx)
	if err != nil {
		panic(err)
	}
}

func (c *AutumnCourt) Receive(actorCtx actor.Context) {
	switch actorCtx.Message().(type) {
	case *actor.Started:
		c.initialize(actorCtx)
	}
}
