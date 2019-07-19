package inventory

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

func TestUnrestrictedAddHandler(t *testing.T) {
	net := network.NewLocalNetwork()

	toTree, err := net.CreateNamedChainTree("toTree")
	require.Nil(t, err)

	previousOwnerKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	objectTree, err := net.CreateNamedChainTree("objectTree")
	require.Nil(t, err)
	objectTreeAuths, err := objectTree.Authentications()
	require.Nil(t, err)
	objectTree, err = net.UpdateChainTree(objectTree, "jasons-game/name", "some obj")
	require.Nil(t, err)
	objectTree, err = net.ChangeChainTreeOwner(objectTree, append(objectTreeAuths, crypto.PubkeyToAddress(previousOwnerKey.PublicKey).String()))
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

	// Ensure transferred object removes previous auths
	toInventoryAuths, err := toInventory.Authentications()
	require.Nil(t, err)
	objectTree, err = net.GetTree(objectTree.MustId())
	require.Nil(t, err)
	newObjectTreeAuths, err := objectTree.Authentications()
	require.Nil(t, err)
	require.Equal(t, toInventoryAuths, newObjectTreeAuths)
}
