package inventory

import (
	"testing"

	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

func TestUnrestrictedAddHandler(t *testing.T) {
	net := network.NewLocalNetwork()

	toTree, err := net.CreateNamedChainTree("toTree")
	require.Nil(t, err)

	objectTree, err := net.CreateNamedChainTree("objectTree")
	require.Nil(t, err)
	objectTree, err = net.UpdateChainTree(objectTree, "jasons-game/name", "some obj")
	require.Nil(t, err)

	toInventory, err := trees.FindInventoryTree(net, toTree.MustId())
	require.Nil(t, err)
	existsBefore, err := toInventory.Exists(objectTree.MustId())
	require.Nil(t, err)
	require.False(t, existsBefore)

	msg := &jasonsgame.TransferredObjectMessage{
		To:     toTree.MustId(),
		Object: objectTree.MustId(),
	}

	h := NewUnrestrictedAddHandler(net)

	require.True(t, h.Supports(msg))
	err = h.Handle(msg)
	require.Nil(t, err)

	toInventory, err = trees.FindInventoryTree(net, msg.To)
	require.Nil(t, err)
	existsAfter, err := toInventory.Exists(objectTree.MustId())
	require.Nil(t, err)
	require.True(t, existsAfter)
}
