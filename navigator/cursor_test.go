package navigator

import (
	"testing"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/storage"
	"github.com/stretchr/testify/require"
)

func TestCursor(t *testing.T) {
	sw := &safewrap.SafeWrap{}

	tree := sw.WrapObject(make(map[string]string))

	chain := sw.WrapObject(make(map[string]string))

	root := sw.WrapObject(map[string]interface{}{
		"chain": chain.Cid(),
		"tree":  tree.Cid(),
		"id":    "test",
	})
	require.Nil(t, sw.Err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	emptyDag, err := dag.NewDagWithNodes(store, root, tree, chain)
	require.Nil(t, err)

	updated, err := emptyDag.SetAsLink([]string{"tree", "data", "jasons-game", "0", "0"}, "hi")
	require.Nil(t, err)
	require.NotNil(t, updated)

	chainTree, err := chaintree.NewChainTree(
		updated,
		nil,
		nil,
	)
	require.Nil(t, err)

	cursor := &cursor{
		chaintree: chainTree,
	}
	output, err := cursor.setLocation(0, 0)
	require.Nil(t, err)
	require.Equal(t, "hi", output)
}
