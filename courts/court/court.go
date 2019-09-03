package court

import (
	"context"
	"path/filepath"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/importer"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

type Court struct {
	ctx  context.Context
	net  network.Network
	name string
	did  string
}

func New(ctx context.Context, net network.Network, name string) *Court {
	return &Court{
		ctx:  ctx,
		net:  net,
		name: name,
	}
}

func (c *Court) Ids() (map[string]interface{}, error) {
	ids, err := c.Resolve([]string{"tree", "data", "ids"})
	if err != nil {
		return nil, err
	}

	if ids == nil {
		return nil, nil
	}

	return ids.(map[string]interface{}), nil
}

func (c *Court) ChainTree() (*consensus.SignedChainTree, error) {
	// caching in order to avoid calculuating key each time
	if c.did == "" {
		tree, err := FindOrCreateNamedTree(c.net, c.name)
		if err != nil {
			return nil, errors.Wrap(err, "fetching court tree")
		}
		c.did = tree.MustId()
		return tree, nil
	}

	return c.net.GetTree(c.did)
}

// Resolve on the court ChainTree
func (c *Court) Resolve(path []string) (interface{}, error) {
	signedTree, err := c.ChainTree()
	if err != nil {
		return nil, err
	}

	val, remaining, err := signedTree.ChainTree.Dag.Resolve(c.ctx, path)

	if err != nil {
		return nil, err
	}

	if len(remaining) > 0 {
		return nil, nil
	}

	return val, nil
}

// Import court chaintress from path
func (c *Court) Import(configPath string) error {
	tree, err := FindOrCreateNamedTree(c.net, c.name)
	if err != nil {
		return errors.Wrap(err, "setting up court tree")
	}

	importIds, err := importer.New(c.net).Import(filepath.Join(configPath, c.name, "import"))
	if err != nil {
		return err
	}

	_, err = c.net.UpdateChainTree(tree, "ids", map[string]interface{}{
		"Locations": importIds.Locations,
		"Objects":   importIds.Objects,
	})

	if err != nil {
		return err
	}
	return nil
}

func SpawnCourt(ctx context.Context, act actor.Actor) {
	actorCtx := actor.EmptyRootContext

	pid := actorCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return act
	}))

	go func() {
		<-ctx.Done()
		actorCtx.Stop(pid)
	}()
}

func FindOrCreateNamedTree(net network.Network, name string) (*consensus.SignedChainTree, error) {
	return net.FindOrCreatePassphraseTree(name)
}
