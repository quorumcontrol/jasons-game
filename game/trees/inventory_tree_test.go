package trees

import (
	"testing"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/stretchr/testify/require"
)

func fakeInventoryTree(t *testing.T, net network.Network) *InventoryTree {
	aTree, err := net.CreateChainTree()
	require.Nil(t, err)
	inventory, err := FindInventoryTree(net, aTree.MustId())
	require.Nil(t, err)
	return inventory
}

func TestInventoryTree_BroadcastTopic(t *testing.T) {
	net := network.NewLocalNetwork()
	inv := fakeInventoryTree(t, net)
	require.Equal(t, inv.BroadcastTopic(), []byte(inv.MustId()+"/inventory"))
}

func TestInventoryObjectCrud(t *testing.T) {
	net := network.NewLocalNetwork()
	inv := fakeInventoryTree(t, net)

	object, err := net.CreateChainTree()
	require.Nil(t, err)
	object, err = net.UpdateChainTree(object, "jasons-game/name", "mighty-sword")
	require.Nil(t, err)

	existsOnInit, err := inv.Exists(object.MustId())
	require.Nil(t, err)
	require.False(t, existsOnInit)

	err = inv.Add(object.MustId())
	require.Nil(t, err)

	existsAfterAdd, err := inv.Exists(object.MustId())
	require.Nil(t, err)
	require.True(t, existsAfterAdd)

	all, err := inv.All()
	require.Nil(t, err)
	require.Equal(t, len(all), 1)
	require.Equal(t, all[object.MustId()], "mighty-sword")

	err = inv.Remove(object.MustId())
	require.Nil(t, err)

	existsAfterRemove, err := inv.Exists(object.MustId())
	require.Nil(t, err)
	require.False(t, existsAfterRemove)

	emptyAll, err := inv.All()
	require.Nil(t, err)
	require.Equal(t, len(emptyAll), 0)
}
