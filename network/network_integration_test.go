// +build integration

package network

import(
	"github.com/stretchr/testify/require"
	"testing"
	"os"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
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
func TestUpdateChainTree(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := "/tmp/test-create-named-tree"
	err := os.MkdirAll(testPath, 0755)
	require.Nil(t,err)
		defer os.RemoveAll(testPath)

	// just to test it doesn't error here
	net := newRemoteNetwork(t, ctx, testPath)

	tree, err := net.CreateNamedChainTree("home")
	require.Nil(t,err)
	tree, err = net.UpdateChainTree(tree, "jasons-game/0/0", &jasonsgame.Location{Description: "hi, welcome"})
	require.Nil(t,err)
	_, err = net.UpdateChainTree(tree, "jasons-game/0/1", &jasonsgame.Location{Description: "north of welcome"})
	require.Nil(t,err)

	newTree,err := net.GetChainTreeByName("home")
	require.Nil(t,err)

	_, err = net.UpdateChainTree(newTree, "jasons-game/0/0", &jasonsgame.Location{Description: "new"})
	require.Nil(t,err)

}
func TestGetChainTreeByName(t *testing.T) {
	err := logging.SetLogLevel("gamenetwork", "debug")
	require.Nil(t, err)
	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := "/tmp/test-get-named-tree"
	err = os.MkdirAll(testPath, 0755)
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
