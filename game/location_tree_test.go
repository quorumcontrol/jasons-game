package game

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLocationTree_SetHandler(t *testing.T) {
	net := network.NewLocalNetwork()

	locationTree, err := net.CreateChainTree()
	require.Nil(t, err)
	location := NewLocationTree(net, locationTree)

	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	handlerTree, err := net.CreateChainTree()
	require.Nil(t, err)
	handlerTree, err = net.ChangeChainTreeOwner(handlerTree, []string{crypto.PubkeyToAddress(key.PublicKey).String()})
	require.Nil(t, err)

	err = location.SetHandler(handlerTree.MustId())
	require.Nil(t, err)

	handler, err := handlers.FindHandlerForTree(net, locationTree.MustId())
	require.Nil(t, err)
	require.Equal(t, handlerTree.MustId(), handler.Did())
}
