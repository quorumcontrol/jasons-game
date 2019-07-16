package game

import (
	"testing"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/stretchr/testify/require"
)

func TestInteractionSetOnTree(t *testing.T) {
	net := network.NewLocalNetwork()
	signedTree, err := net.CreateChainTree()
	require.Nil(t, err)
	tree := NewLocationTree(net, signedTree)

	interaction := &RespondInteraction{
		Command:  "marco",
		Response: "polo",
	}

	err = tree.AddInteraction(interaction)
	require.Nil(t, err)

	list, err := tree.InteractionsList()
	require.Nil(t, err)
	require.Equal(t, len(list), 1)

	storedInteraction, ok := list[0].(*RespondInteraction)
	require.True(t, ok)
	require.Equal(t, storedInteraction.Command, interaction.Command)
	require.Equal(t, storedInteraction.Response, interaction.Response)
}

func TestCipherInteraction(t *testing.T) {
	cmd := "whisper to the wall"
	secret := "sherbert lemon"

	si := &RespondInteraction{
		Response: "success response",
	}
	fi := &RespondInteraction{
		Response: "failure response",
	}

	ci, err := NewCipherInteraction(cmd, secret, si, fi)
	require.Nil(t, err)

	interaction, didUnseal, err := ci.Unseal("bad secret")
	require.Nil(t, err)
	require.False(t, didUnseal)
	require.Equal(t, fi.Response, interaction.(*RespondInteraction).Response)

	interaction, didUnseal, err = ci.Unseal(secret)
	require.Nil(t, err)
	require.True(t, didUnseal)
	require.Equal(t, si.Response, interaction.(*RespondInteraction).Response)
}
