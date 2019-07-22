package importer

import (
	"context"
	"testing"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/stretchr/testify/require"
)

func TestImport(t *testing.T) {
	ctx := context.Background()
	net := network.NewLocalNetwork()
	path := "import-example"
	ids, err := New(net).Import(path)
	require.Nil(t, err)

	tree, err := net.GetTree(ids.Locations["home"])
	require.Nil(t, err)
	require.NotNil(t, tree)
	val, _, err := tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "description"})
	require.Nil(t, err)
	require.Equal(t, val, "you have entered the world of the fairies, in front of you sits a great forest")
	val, _, err = tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "interactions"})
	require.Nil(t, err)
	require.Equal(t, len(val.(map[string]interface{})), 1)
	val, _, err = tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "inventory"})
	require.Nil(t, err)
	require.Equal(t, len(val.(map[string]interface{})), 1)

	tree, err = net.GetTree(ids.Locations["forest"])
	require.Nil(t, err)
	require.NotNil(t, tree)
	val, _, err = tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "description"})
	require.Nil(t, err)
	require.Equal(t, val, "you are now in the forest, what now?")
	val, _, err = tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "interactions"})
	require.Nil(t, err)
	require.Equal(t, len(val.(map[string]interface{})), 2)
	val, _, err = tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "inventory"})
	require.Nil(t, err)
	require.Nil(t, val)

	tree, err = net.GetTree(ids.Objects["idol"])
	require.Nil(t, err)
	require.NotNil(t, tree)
	val, _, err = tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "description"})
	require.Nil(t, err)
	require.Equal(t, val, "this is an idol")
	val, _, err = tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "interactions"})
	require.Nil(t, err)
	require.Equal(t, len(val.(map[string]interface{})), 4)
}
