package network

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	blockservice "github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/ipfs/go-merkledag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicTreeStore(t *testing.T) {
	keystore := datastore.NewMapDatastore()

	bstore := blockstore.NewBlockstore(keystore)
	bserv := blockservice.New(bstore, offline.Exchange(bstore))
	dag := merkledag.NewDAGService(bserv)

	ipldstore := NewIPLDTreeStore(dag, keystore)
	SubtestAll(t, ipldstore)
	SubtestTreeStore(t, ipldstore)
}

func SubtestTreeStore(t *testing.T, ts TreeStore) {
	t.Run("GetTree", func(t *testing.T) { SubtestGetTree(t, ts) })
	t.Run("SaveTreeMetadata", func(t *testing.T) { SubtestSaveTreeMetadata(t, ts) })
}

func SubtestGetTree(t *testing.T, ts TreeStore) {
	tree := createTree(t, ts)
	err := ts.SaveTreeMetadata(tree)
	require.Nil(t, err)
	reconstituted, err := ts.GetTree(tree.MustId())
	require.Nil(t, err)

	treeData, _, err := tree.ChainTree.Dag.Resolve([]string{"tree", "data"})
	require.Nil(t, err)
	reconstitutedData, _, err := reconstituted.ChainTree.Dag.Resolve([]string{"tree", "data"})
	require.Nil(t, err)

	require.Equal(t, treeData, reconstitutedData)
}

func SubtestSaveTreeMetadata(t *testing.T, ts TreeStore) {
	tree := createTree(t, ts)
	err := ts.SaveTreeMetadata(tree)
	require.Nil(t, err)
}

func createTree(t *testing.T, ts TreeStore) *consensus.SignedChainTree {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	tree, err := consensus.NewSignedChainTree(key.PublicKey, ts)
	require.Nil(t, err)

	updated, err := tree.ChainTree.Dag.SetAsLink([]string{"tree", "data", "jasons-game", "0", "0"}, &navigator.Location{Description: "hi, welcome"})
	require.Nil(t, err)

	updated, err = updated.SetAsLink([]string{"tree", "data", "jasons-game", "0", "1"}, &navigator.Location{Description: "you are north of the welcome"})
	require.Nil(t, err)
	tree.ChainTree.Dag = updated
	return tree
}

// This is just copypasta from the chaintree repo
func SubtestAll(t *testing.T, ns nodestore.NodeStore) {
	t.Run("CreateNode", func(t *testing.T) { SubtestInterfaceCreateNode(t, ns) })
	t.Run("GetNode", func(t *testing.T) { SubtestInterfaceGetNode(t, ns) })
	t.Run("DeleteNode", func(t *testing.T) { SubtestInterfaceDeleteNode(t, ns) })
	t.Run("DeleteTree", func(t *testing.T) { SubtestInterfaceDeleteTree(t, ns) })
	t.Run("Resolve", func(t *testing.T) { SubtestInterfaceResolve(t, ns) })
}

func SubtestInterfaceCreateNode(t *testing.T, ns nodestore.NodeStore) {
	obj := map[string]string{"createnode": "value"}
	sw := &safewrap.SafeWrap{}
	testNode := sw.WrapObject(obj)

	node, err := ns.CreateNode(obj)
	require.Nil(t, err)
	assert.Equal(t, testNode.Cid().String(), node.String())
}

func SubtestInterfaceGetNode(t *testing.T, ns nodestore.NodeStore) {
	obj := map[string]string{"getnode": "value"}
	sw := &safewrap.SafeWrap{}
	testNode := sw.WrapObject(obj)

	_, err := ns.CreateNode(obj)
	require.Nil(t, err)

	node, err := ns.GetNode(testNode.Cid())
	require.Nil(t, err)
	assert.Equal(t, testNode.Cid().String(), node.String())
}

func SubtestInterfaceDeleteNode(t *testing.T, ns nodestore.NodeStore) {
	type testStruct struct {
		description  string
		setup        func(t *testing.T) cid.Cid
		shouldErr    bool
		shouldDelete bool
	}
	for _, test := range []testStruct{
		{
			description:  "an unreferenced node",
			shouldErr:    false,
			shouldDelete: true,
			setup: func(t *testing.T) cid.Cid {
				defaultMap := map[string]string{
					"test1": "value1",
				}
				node, err := ns.CreateNode(defaultMap)
				require.Nil(t, err)
				return node.Cid()
			},
		},
		{
			description:  "a referenced node",
			shouldErr:    false,
			shouldDelete: true,
			setup: func(t *testing.T) cid.Cid {
				defaultMap := map[string]string{
					"test2": "value2",
				}
				node, err := ns.CreateNode(defaultMap)
				require.Nil(t, err)
				root := map[string]cid.Cid{
					"ref": node.Cid(),
				}
				_, err = ns.CreateNode(root)
				require.Nil(t, err)

				return node.Cid()
			},
		},
		{
			description:  "a node with link",
			shouldErr:    false,
			shouldDelete: true,
			setup: func(t *testing.T) cid.Cid {
				defaultMap := map[string]string{
					"test3": "value3",
				}
				node, err := ns.CreateNode(defaultMap)
				require.Nil(t, err)
				root := map[string]cid.Cid{
					"ref": node.Cid(),
				}
				rootNode, err := ns.CreateNode(root)
				require.Nil(t, err)

				return rootNode.Cid()
			},
		},
	} {
		toDelete := test.setup(t)
		err := ns.DeleteNode(toDelete)
		require.Nil(t, err, test.description)
		existing, err := ns.GetNode(toDelete)
		require.Nil(t, err, test.description)

		if test.shouldDelete {
			require.Nil(t, existing, test.description)
		} else {
			require.NotNil(t, existing, test.description)
		}
		if test.shouldErr {
			require.NotNil(t, err, test.description)
		} else {
			require.Nil(t, err, test.description)
		}

	}
}

func SubtestInterfaceDeleteTree(t *testing.T, ns nodestore.NodeStore) {
	type testCase struct {
		description string
		setup       func() (nodesToCreate []*cbornode.Node, tipToDelete cid.Cid)
		tests       func(nodes []*cbornode.Node)
		shouldErr   bool
	}

	defaultMap := map[string]string{
		"hi": "hi",
	}

	for _, tc := range []testCase{
		{
			description: "a single node",
			shouldErr:   false,
			setup: func() (nodesToCreate []*cbornode.Node, tipToDelete cid.Cid) {
				sw := safewrap.SafeWrap{}
				node := sw.WrapObject(defaultMap)
				return []*cbornode.Node{node}, node.Cid()
			},
			tests: func(nodes []*cbornode.Node) {
				saved, err := ns.GetNode(nodes[0].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)
			},
		},
		{
			description: "a tree",
			shouldErr:   false,
			setup: func() (nodesToCreate []*cbornode.Node, tipToDelete cid.Cid) {
				sw := safewrap.SafeWrap{}
				node := sw.WrapObject(defaultMap)
				root := map[string]cid.Cid{"child": node.Cid()}
				rootNode := sw.WrapObject(root)

				return []*cbornode.Node{node, rootNode}, rootNode.Cid()
			},
			tests: func(nodes []*cbornode.Node) {
				saved, err := ns.GetNode(nodes[0].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)
				saved, err = ns.GetNode(nodes[1].Cid())
				assert.Nil(t, err)
				assert.Nil(t, saved)
			},
		},
	} {
		nodes, tipToDelete := tc.setup()
		for _, node := range nodes {
			_, err := ns.CreateNodeFromBytes(node.RawData())
			require.Nil(t, err)
		}
		err := ns.DeleteTree(tipToDelete)
		if tc.shouldErr {
			assert.NotNil(t, err, tc.description)
		} else {
			assert.Nil(t, err, tc.description)
		}
		tc.tests(nodes)
	}
}

func SubtestInterfaceResolve(t *testing.T, ns nodestore.NodeStore) {
	sw := &safewrap.SafeWrap{}
	child := sw.WrapObject(map[string]interface{}{
		"name": "child",
	})

	root := sw.WrapObject(map[string]interface{}{
		"child": child.Cid(),
		"key":   "value",
	})

	assert.Nil(t, sw.Err)
	_, err := ns.CreateNodeFromBytes(child.RawData())
	require.Nil(t, err)
	_, err = ns.CreateNodeFromBytes(root.RawData())
	require.Nil(t, err)

	// Resolves through the tree
	val, remaining, err := ns.Resolve(root.Cid(), []string{"child", "name"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "child", val)

	// Resolves on the object itself
	val, remaining, err = ns.Resolve(root.Cid(), []string{"key"})
	assert.Nil(t, err)
	assert.Empty(t, remaining)
	assert.Equal(t, "value", val)

	// Does not error on missing paths, but returns a nil value, with the part of the path missing
	val, remaining, err = ns.Resolve(root.Cid(), []string{"child", "missing", "path"})
	assert.Nil(t, err)
	assert.Nil(t, val)
	assert.Equal(t, []string{"missing", "path"}, remaining)
}
