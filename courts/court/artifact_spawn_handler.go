package court

import (
	"fmt"
	"path/filepath"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/courts/artifact"
	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/handlers/inventory"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

const artifactSpawnFile = "artifact_spawn_config.yml"

type SpawnConfig struct {
	Locations []string
	Forgers   []string
}

type ArtifactSpawnHandlerConfig struct {
	Court      *Court
	ConfigPath string
}

type ArtifactSpawnHandler struct {
	*inventory.UnrestrictedRemoveHandler

	court *Court
	net   network.Network
	tree  *consensus.SignedChainTree
	cfg   *SpawnConfig
}

func NewArtifactSpawnHandler(config *ArtifactSpawnHandlerConfig) (*ArtifactSpawnHandler, error) {
	handler := &ArtifactSpawnHandler{
		court: config.Court,
		net:   config.Court.Network(),
		UnrestrictedRemoveHandler: inventory.NewUnrestrictedRemoveHandler(config.Court.Network()),
	}
	err := handler.setup(config.ConfigPath)
	if err != nil {
		return nil, err
	}
	return handler, nil
}

func (h *ArtifactSpawnHandler) Tree() *consensus.SignedChainTree {
	return h.tree
}

func (h *ArtifactSpawnHandler) Name() string {
	return h.court.Name() + "-artifact-spawn-handler"
}

func (h *ArtifactSpawnHandler) setup(configPath string) error {
	spawnConfigPath := filepath.Join(configPath, h.court.Name(), artifactSpawnFile)

	vars, err := h.court.Ids()
	if err != nil {
		return errors.Wrap(err, "error fetching ids for court")
	}

	h.cfg = &SpawnConfig{}
	err = config.ReadYaml(spawnConfigPath, h.cfg, vars)
	if err != nil {
		return errors.Wrap(err, "error processing "+spawnConfigPath)
	}

	if len(h.cfg.Locations) == 0 {
		return errors.Wrap(err, "must set 1 or more .locations in "+spawnConfigPath)
	}

	if len(h.cfg.Forgers) == 0 {
		return errors.Wrap(err, "must set 1 or more .forgers in "+spawnConfigPath)
	}

	h.tree, err = h.net.FindOrCreatePassphraseTree(h.Name())
	if err != nil {
		return err
	}
	handlerDid := h.tree.MustId()

	for _, spawnLocation := range h.cfg.Locations {
		locTree, err := h.net.GetTree(spawnLocation)
		if err != nil {
			return errors.Wrap(err, "getting location tree "+spawnLocation)
		}

		locationHandler, err := handlers.FindHandlerForTree(h.net, spawnLocation)
		if err != nil {
			return errors.Wrap(err, "checking location handler "+spawnLocation)
		}

		if locationHandler != nil && locationHandler.Did() != handlerDid {
			return fmt.Errorf("location %s already has a handler attached, cannot use artifact spawner here", spawnLocation)
		}

		if locationHandler == nil {
			loc := game.NewLocationTree(h.net, locTree)

			err = loc.SetHandler(handlerDid)
			if err != nil {
				return errors.Wrap(err, "setting location handler "+spawnLocation)
			}
		}
	}

	respawner, err := artifact.NewRespawnActor(h.court.ctx, &artifact.RespawnActorConfig{
		Network:    h.net,
		Locations:  h.cfg.Locations,
		Forgers:    h.cfg.Forgers,
		ConfigPath: configPath,
	})
	if err != nil {
		return errors.Wrap(err, "creating respawn actor")
	}

	actorCtx := actor.EmptyRootContext
	respawner.Start(actorCtx)
	pid := respawner.PID()

	go func() {
		<-h.court.ctx.Done()
		actorCtx.Stop(pid)
	}()

	return nil
}
