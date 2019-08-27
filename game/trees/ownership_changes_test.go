package trees

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/jasons-game/network"
)

func TestOwnershipChanges(t *testing.T) {
	net := network.NewLocalNetwork()
	netKeyAddr := crypto.PubkeyToAddress(*net.PublicKey()).String()

	count := 3
	expectedAddrs := make([][]string, count)

	tree, err := net.CreateChainTree()
	require.Nil(t, err)

	for i := 0; i < count; i++ {
		key, err := crypto.GenerateKey()
		require.Nil(t, err)
		expectedAddrs[i] = []string{
			netKeyAddr,
			crypto.PubkeyToAddress(key.PublicKey).String(),
		}

		tree, err = net.UpdateChainTree(tree, "test", i)
		require.Nil(t, err)

		tree, err = net.ChangeChainTreeOwner(tree, expectedAddrs[i])
		require.Nil(t, err)
	}

	changes, err := OwnershipChanges(context.Background(), tree.ChainTree)
	require.Nil(t, err)
	require.Len(t, changes, count+1) // +1 for the original change by network

	for i, change := range changes[0:count] {
		assert.ElementsMatch(t, change.Authentications, expectedAddrs[count-1-i])
	}
}
