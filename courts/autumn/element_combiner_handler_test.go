package autumn

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/network"
)

func TestElementCombinerHandler(t *testing.T) {
	net := network.NewLocalNetwork()

	playerTree, err := net.CreateNamedChainTree("playerTree")
	require.Nil(t, err)

	serviceTree, err := net.CreateNamedChainTree("serviceTree")
	require.Nil(t, err)

	cfg := &AutumnConfig{}
	err = config.ReadYaml("../yml-test/autumn/config.yml", cfg)
	require.Nil(t, err)

	h, err := NewElementCombinerHandler(&ElementCombinerHandlerConfig{
		Network:      net,
		Name:         "weaver",
		Location:     serviceTree.MustId(),
		Elements:     cfg.Elements,
		Combinations: cfg.Weaver,
	})
	require.Nil(t, err)

	client := &MockElementClient{
		Net:     net,
		H:       h,
		Player:  playerTree.MustId(),
		Service: serviceTree.MustId(),
	}

	t.Run("with a correct combination", func(t *testing.T) {
		require.Nil(t, client.Send(100))
		require.True(t, client.HasBowl())

		require.Nil(t, client.Send(200))
		require.True(t, client.HasBowl())

		msg, err := client.PickupBowl()
		require.Nil(t, err)
		require.Empty(t, msg.Error)

		require.True(t, client.HasElement(300))
	})

	t.Run("with an incorrect combination", func(t *testing.T) {
		require.Nil(t, client.Send(100))
		require.True(t, client.HasBowl())

		require.Nil(t, client.Send(101))
		require.True(t, client.HasBowl())

		msg, _ := client.PickupBowl()
		require.NotNil(t, msg)
		require.Equal(t, msg.Error, fmt.Sprintf(combinationFailureMsg, "Weaver"))
		require.False(t, client.HasBowl())
	})

	t.Run("with a blocked combination", func(t *testing.T) {
		require.Nil(t, client.Send(101))
		require.True(t, client.HasBowl())

		require.Nil(t, client.Send(102))
		require.True(t, client.HasBowl())

		msg, _ := client.PickupBowl()
		require.NotNil(t, msg)
		require.Equal(t, msg.Error, combinationBlockedFailureMsg)
		require.False(t, client.HasBowl())
	})

	t.Run("with too few elements", func(t *testing.T) {
		require.Nil(t, client.Send(100))
		require.True(t, client.HasBowl())

		msg, _ := client.PickupBowl()
		require.NotNil(t, msg)
		require.Equal(t, msg.Error, fmt.Sprintf(combinationNumFailureMsg, 2))
		require.True(t, client.HasBowl())
	})
}


