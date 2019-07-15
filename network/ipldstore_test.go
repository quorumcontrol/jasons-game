package network

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	blockservice "github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/stretchr/testify/require"
)

func newTestIpldTreeStore() *IPLDTreeStore {
	keystore := datastore.NewMapDatastore()
	bstore := blockstore.NewBlockstore(keystore)
	bserv := blockservice.New(bstore, offline.Exchange(bstore))
	dag := merkledag.NewDAGService(bserv)
	return NewIPLDTreeStore(dag, keystore, new(DevNullTipGetter))
}

func createTree(t *testing.T, ts TreeStore) *consensus.SignedChainTree {
	ctx := context.TODO()

	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	tree, err := consensus.NewSignedChainTree(key.PublicKey, ts)
	require.Nil(t, err)

	updated, err := tree.ChainTree.Dag.SetAsLink(ctx, []string{"tree", "data", "jasons-game", "0", "0"}, &jasonsgame.Location{Description: "hi, welcome"})
	require.Nil(t, err)

	updated, err = updated.SetAsLink(ctx, []string{"tree", "data", "jasons-game", "0", "1"}, &jasonsgame.Location{Description: "you are north of the welcome"})
	require.Nil(t, err)
	tree.ChainTree.Dag = updated
	return tree
}

func TestGetTree(t *testing.T) {
	ctx := context.TODO()
	ts := newTestIpldTreeStore()

	tree := createTree(t, ts)
	err := ts.SaveTreeMetadata(tree)
	require.Nil(t, err)
	reconstituted, err := ts.GetTree(tree.MustId())
	require.Nil(t, err)

	treeData, _, err := tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data"})
	require.Nil(t, err)
	reconstitutedData, _, err := reconstituted.ChainTree.Dag.Resolve(ctx, []string{"tree", "data"})
	require.Nil(t, err)

	require.Equal(t, treeData, reconstitutedData)
}

func TestIpldStore(t *testing.T) {
	ctx := context.TODO()
	ts := newTestIpldTreeStore()

	sw := safewrap.SafeWrap{}
	obj := map[string]string{"test": "test"}
	n := sw.WrapObject(obj)
	require.Nil(t, sw.Err)

	err := ts.Add(ctx, n)
	require.Nil(t, err)

	returnedNode, err := ts.Get(ctx, n.Cid())
	require.Nil(t, err)
	require.NotNil(t, returnedNode)

	// works with a missing node

	obj = map[string]string{"test": "diff"}
	n = sw.WrapObject(obj)
	require.Nil(t, sw.Err)

	_, err = ts.Get(ctx, n.Cid())
	require.Equal(t, format.ErrNotFound, err)
}
