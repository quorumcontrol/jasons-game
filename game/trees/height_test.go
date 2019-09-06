package trees

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/jasons-game/network"
)

func TestHeight(t *testing.T) {
	ctx := context.Background()
	net := network.NewLocalNetwork()

	tree, err := net.CreateChainTree()
	require.Nil(t, err)

	tree, err = net.UpdateChainTree(tree, "test", "1")
	require.Nil(t, err)

	require.Equal(t, MustHeight(ctx, tree.ChainTree), uint64(1))

	tree, err = net.UpdateChainTree(tree, "test", "2")
	require.Nil(t, err)

	require.Equal(t, MustHeight(ctx, tree.ChainTree), uint64(2))
	height, err := Height(ctx, tree.ChainTree)
	require.Nil(t, err)
	require.Equal(t, height, uint64(2))
}

func TestAtHeight(t *testing.T) {
	ctx := context.Background()
	net := network.NewLocalNetwork()

	tree, err := net.CreateChainTree()
	require.Nil(t, err)

	tree, err = net.UpdateChainTree(tree, "test", "1")
	require.Nil(t, err)

	tree, err = net.UpdateChainTree(tree, "test", "2")
	require.Nil(t, err)

	tree, err = net.UpdateChainTree(tree, "test", "3")
	require.Nil(t, err)

	treeAt, err := AtHeight(ctx, tree.ChainTree, 1)
	require.Nil(t, err)
	val, rem, err := treeAt.Dag.Resolve(ctx, []string{"tree", "data", "test"})
	require.Nil(t, err)
	require.Nil(t, rem)
	require.Equal(t, val, "1")

	treeAt, err = AtHeight(ctx, tree.ChainTree, 2)
	require.Nil(t, err)
	val, rem, err = treeAt.Dag.Resolve(ctx, []string{"tree", "data", "test"})
	require.Nil(t, err)
	require.Nil(t, rem)
	require.Equal(t, val, "2")

	treeAt, err = AtHeight(ctx, tree.ChainTree, 3)
	require.Nil(t, err)
	val, rem, err = treeAt.Dag.Resolve(ctx, []string{"tree", "data", "test"})
	require.Nil(t, err)
	require.Nil(t, rem)
	require.Equal(t, val, "3")

	_, err = AtHeight(ctx, tree.ChainTree, 4)
	require.NotNil(t, err)
}
