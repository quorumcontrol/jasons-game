package spring

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	logging "github.com/ipfs/go-log"

	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/importer"

	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("spring")

type springConfig struct {
	Stages        map[string]*importer.ImportLocation `yaml:"stages"`
	StageRotation map[int]string                      `yaml:"stage_rotation"`
}

type SpringCourt struct {
	ctx        context.Context
	net        network.Network
	court      *court.Court
	configPath string
}

func New(ctx context.Context, net network.Network, configPath string) *SpringCourt {
	return &SpringCourt{
		ctx:        ctx,
		net:        net,
		court:      court.New(ctx, net, "spring"),
		configPath: configPath,
	}
}

func (c *SpringCourt) Start() {
	court.SpawnCourt(c.ctx, c)
}

func (c *SpringCourt) config() *springConfig {
	ids, err := c.court.Ids()
	if err != nil {
		panic(errors.Wrap(err, "error fetching court ids"))
	}

	cfg := &springConfig{}
	err = config.ReadYaml(filepath.Join(c.configPath, "spring/config.yml"), cfg, ids)
	if err != nil {
		panic(errors.Wrap(err, "error fetching config"))
	}

	if len(cfg.StageRotation) != 24 {
		panic("stage rotation should have 24 elements, one for each hour of the day")
	}
	for i := 0; i < 24; i++ {
		if _, ok := cfg.StageRotation[i]; !ok {
			panic(fmt.Sprintf("stage rotation should have 24 elements, one for each hour of the day - missing index %d", i))
		}
	}

	return cfg
}

func (c *SpringCourt) updateTimeStage() error {
	cfg := c.config()

	currentHour := time.Now().UTC().Hour()
	currentStage := cfg.StageRotation[currentHour]

	currentStageCfg, ok := cfg.Stages[currentStage]
	if !ok {
		return fmt.Errorf("stage %s not found", currentStage)
	}

	timeTreeDid, err := c.court.Resolve([]string{"tree", "data", "ids", "Locations", "stageTimeFields"})
	if err != nil || timeTreeDid == nil {
		return fmt.Errorf("stage time fields not found")
	}

	err = importer.New(c.net).UpdateLocation(timeTreeDid.(string), currentStageCfg)
	if err != nil {
		return err
	}

	return nil
}

func (c *SpringCourt) initialize(actorCtx actor.Context) {
	err := c.court.Import(c.configPath)
	if err != nil {
		panic(err)
	}

	err = c.updateTimeStage()
	if err != nil {
		panic(err)
	}

	// run update time stage every hour
	go func() {
		for {
			nextHour := time.Until(time.Now().Truncate(time.Hour).Add(time.Hour))
			time.Sleep(nextHour)
			err := c.updateTimeStage()
			if err != nil {
				log.Error(err)
			}
		}
	}()
}

func (c *SpringCourt) Receive(actorCtx actor.Context) {
	switch actorCtx.Message().(type) {
	case *actor.Started:
		c.initialize(actorCtx)
	}
}
