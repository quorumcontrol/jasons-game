package summer

import (
	"context"
	"crypto/ecdsa"
	"path/filepath"

	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	logging "github.com/ipfs/go-log"

	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/importer"

	"github.com/quorumcontrol/jasons-game/courts/artifact"
	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/game"
	handlers "github.com/quorumcontrol/jasons-game/handlers"
	inventoryHandlers "github.com/quorumcontrol/jasons-game/handlers/inventory"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/service"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
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
	treeKey    *ecdsa.PrivateKey
	configPath string
}

func New(ctx context.Context, net network.Network, configPath string) *SummerCourt {
	return &SummerCourt{
		ctx:        ctx,
		net:        net,
		configPath: configPath,
	}
}

func (c *SummerCourt) Start() {
	actorCtx := actor.EmptyRootContext

	pid := actorCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return c
	}))

	go func() {
		<-c.ctx.Done()
		actorCtx.Stop(pid)
	}()
}

func (c *SummerCourt) ids() (map[string]interface{}, error) {
	ids, _, err := c.chainTree().ChainTree.Dag.Resolve(c.ctx, []string{"tree", "data", "ids"})
	if err != nil {
		return nil, err
	}

	if ids == nil {
		return nil, nil
	}

	return ids.(map[string]interface{}), nil
}

func (c *SummerCourt) chainTree() *consensus.SignedChainTree {
	var err error

	if c.treeKey == nil {
		c.treeKey, err = consensus.PassPhraseKey(crypto.FromECDSA(c.net.PrivateKey()), []byte("summercourt"))
		if err != nil {
			panic(errors.Wrap(err, "setting up court keys"))
		}
	}

	tree, err := c.net.GetTree(consensus.EcdsaPubkeyToDid(c.treeKey.PublicKey))
	if err != nil {
		panic(errors.Wrap(err, "getting court chaintree"))
	}

	return tree
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
	handlerTree, err := findOrCreateNamedTree(c.net, artifactPickupHandlerTreeName)
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
	handlerTree, err := findOrCreateNamedTree(c.net, prizeHandlerTreeName)
	if err != nil {
		panic(err)
	}
	handler, err := NewSummerPrizeHandler(c)
	if err != nil {
		panic(err)
	}
	servicePID, err := actorCtx.SpawnNamed(service.NewServiceActorPropsWithTree(c.net, handler, handlerTree), prizeHandlerTreeName)
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
	var err error
	tree := c.chainTree()

	// tree is empty, import
	if tree == nil {
		tree, err = c.net.CreateChainTreeWithKey(c.treeKey)
		if err != nil {
			panic(errors.Wrap(err, "setting up court chaintree"))
		}
		_, err = c.net.ChangeChainTreeOwnerWithKey(tree, c.treeKey, []string{
			crypto.PubkeyToAddress(c.treeKey.PublicKey).String(),
			crypto.PubkeyToAddress(*c.net.PublicKey()).String(),
		})
		if err != nil {
			panic(errors.Wrap(err, "chowning court chaintree"))
		}

		importIds, err := importer.New(c.net).Import(filepath.Join(c.configPath, "summer/import"))
		if err != nil {
			panic(err)
		}

		_, err = c.net.UpdateChainTree(c.chainTree(), "ids", map[string]interface{}{
			"Locations": importIds.Locations,
			"Objects":   importIds.Objects,
		})

		if err != nil {
			panic(err)
		}
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

func findOrCreateNamedTree(net network.Network, name string) (*consensus.SignedChainTree, error) {
	treeKey, err := consensus.PassPhraseKey(crypto.FromECDSA(net.PrivateKey()), []byte(name))
	if err != nil {
		return nil, errors.Wrap(err, "setting up named tree keys")
	}

	tree, err := net.GetTree(consensus.EcdsaPubkeyToDid(treeKey.PublicKey))
	if err != nil {
		return nil, errors.Wrap(err, "getting named chaintree")
	}

	if tree == nil {
		tree, err = net.CreateChainTreeWithKey(treeKey)
		if err != nil {
			return nil, errors.Wrap(err, "setting up named chaintree")
		}

		tree, err = net.ChangeChainTreeOwnerWithKey(tree, treeKey, []string{
			crypto.PubkeyToAddress(treeKey.PublicKey).String(),
			crypto.PubkeyToAddress(*net.PublicKey()).String(),
		})

		if err != nil {
			return nil, errors.Wrap(err, "chowning court chaintree")
		}
	}

	return tree, nil
}
