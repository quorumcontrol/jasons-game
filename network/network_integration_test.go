// +build integration

package network

import(
	"github.com/stretchr/testify/require"
	"testing"
	"os"
	logging "github.com/ipfs/go-log"

	"context"
)

func newRemoteNetwork(t *testing.T, ctx context.Context, path string) Network {
	group,err := setupNotaryGroup(ctx)
	require.Nil(t,err)

	net,err := NewRemoteNetwork(ctx, group, path)
	require.Nil(t,err)
	return net
}

func TestNewNetwork(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := "/tmp/test-new-network"
	err := os.MkdirAll(testPath, 0755)
	require.Nil(t,err)
	defer os.RemoveAll(testPath)

	// just to test it doesn't error here
	newRemoteNetwork(t, ctx, testPath)
}
func TestCreateNamedChainTree(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := "/tmp/test-create-named-tree"
	err := os.MkdirAll(testPath, 0755)
	require.Nil(t,err)
		defer os.RemoveAll(testPath)

	// just to test it doesn't error here
	net := newRemoteNetwork(t, ctx, testPath)
	_,err = net.CreateNamedChainTree("test-create-named-tree")
	require.Nil(t,err)
}
func TestGetChainTreeByName(t *testing.T) {
	logging.SetLogLevel("gamenetwork", "debug")
	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := "/tmp/test-get-named-tree"
	err := os.MkdirAll(testPath, 0755)
	require.Nil(t,err)
		defer os.RemoveAll(testPath)

	// just to test it doesn't error here
	log.Infof("new remote network")
	net := newRemoteNetwork(t, ctx, testPath)
	log.Infof("before create network")
	tree,err := net.CreateNamedChainTree("test-get-named-tree")
	require.Nil(t,err)
	log.Infof("after create network")

	lookupTree,err := net.GetChainTreeByName("test-get-named-tree")
	require.Nil(t,err)
	log.Infof("after get chaintree")

	require.Equal(t, tree.MustId(), lookupTree.MustId())
}