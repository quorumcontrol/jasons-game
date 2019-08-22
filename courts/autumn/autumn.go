package autumn

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

	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/service"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var log = logging.Logger("autumn")

type autumnConfig struct {
	Elements []*element            `yaml:"elements"`
	Binder   []*elementCombination `yaml:"binder"`
	Weaver   []*elementCombination `yaml:"weaver"`
}

type AutumnCourt struct {
	ctx        context.Context
	net        network.Network
	treeKey    *ecdsa.PrivateKey
	configPath string
}

func New(ctx context.Context, net network.Network, configPath string) *AutumnCourt {
	return &AutumnCourt{
		ctx:        ctx,
		net:        net,
		configPath: configPath,
	}
}

func (c *AutumnCourt) Start() {
	actorCtx := actor.EmptyRootContext

	pid := actorCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return c
	}))

	go func() {
		<-c.ctx.Done()
		actorCtx.Stop(pid)
	}()
}

func (c *AutumnCourt) ids() (map[string]interface{}, error) {
	ids, _, err := c.chainTree().ChainTree.Dag.Resolve(c.ctx, []string{"tree", "data", "ids"})
	if err != nil {
		return nil, err
	}

	if ids == nil {
		return nil, nil
	}

	return ids.(map[string]interface{}), nil
}

func (c *AutumnCourt) chainTree() *consensus.SignedChainTree {
	var err error

	if c.treeKey == nil {
		c.treeKey, err = consensus.PassPhraseKey(crypto.FromECDSA(c.net.PrivateKey()), []byte("autumncourt"))
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

func (c *AutumnCourt) config() *autumnConfig {
	ids, err := c.ids()
	if err != nil {
		panic(errors.Wrap(err, "error fetching court ids"))
	}

	cfg := &autumnConfig{}
	err = config.ReadYaml(filepath.Join(c.configPath, "autumn/config.yml"), cfg, ids)
	if err != nil {
		panic(errors.Wrap(err, "error fetching config"))
	}

	return cfg
}

func (c *AutumnCourt) setupCombinationHandler(actorCtx actor.Context, name string, elements []*element, combinations []*elementCombination) {
	locationDidUncast, _, err := c.chainTree().ChainTree.Dag.Resolve(c.ctx, []string{"tree", "data", "ids", "Locations", name})
	if err != nil {
		panic(err)
	}
	locationDid, ok := locationDidUncast.(string)
	if !ok {
		panic("Could not find location for " + name)
	}

	handlerName := name + "-handler"
	handlerTree, err := findOrCreateNamedTree(c.net, handlerName)
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

		importIds, err := importer.New(c.net).Import(filepath.Join(c.configPath, "autumn/import"))
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
	c.setupCombinationHandler(actorCtx, "weaver", config.Elements, config.Weaver)
	c.setupCombinationHandler(actorCtx, "binder", config.Elements, config.Binder)
}

func (c *AutumnCourt) Receive(actorCtx actor.Context) {
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
