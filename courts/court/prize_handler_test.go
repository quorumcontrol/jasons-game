package court

import (
	"context"
	"testing"

	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

func TestSummerCourt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	net := network.NewLocalNetwork()

	court := New(ctx, net, "prize-test")
	courtTree, err := court.ChainTree()
	require.Nil(t, err)

	playerTree, err := net.CreateLocalChainTree("player")
	require.Nil(t, err)

	locTree, err := net.CreateChainTree()
	require.Nil(t, err)
	_, err = net.UpdateChainTree(courtTree, "ids", map[string]interface{}{"Locations": map[string]string{"loc1": locTree.MustId()}})
	require.Nil(t, err)

	var validatorCalled bool
	var cleanupCalled bool
	validatorShouldPass := true

	handler, err := NewPrizeHandler(&PrizeHandlerConfig{
		Court:           court,
		PrizeConfigPath: "../yml-test/court/prize_config.yml",
		ValidatorFunc: func(_ *jasonsgame.RequestObjectTransferMessage) (bool, error) {
			validatorCalled = true
			return validatorShouldPass, nil
		},
		CleanupFunc: func(_ *jasonsgame.RequestObjectTransferMessage) error {
			cleanupCalled = true
			return nil
		},
	})
	require.Nil(t, err)

	objDid, err := handler.currentObjectDid()
	require.Nil(t, err)

	spawnedObj, err := game.FindObjectTree(net, objDid)
	require.Nil(t, err)

	spawnedName, err := spawnedObj.GetName()
	require.Equal(t, spawnedName, "spawn-obj")
	require.Nil(t, err)

	err = handler.Handle(&jasonsgame.RequestObjectTransferMessage{
		From:   locTree.MustId(),
		To:     playerTree.MustId(),
		Object: objDid,
	})
	require.Nil(t, err)

	require.True(t, validatorCalled)
	require.True(t, cleanupCalled)

	prizeObj, err := game.FindObjectTree(net, objDid)
	require.Nil(t, err)

	prizeName, err := prizeObj.GetName()
	require.Equal(t, prizeName, "test-prize")
	require.Nil(t, err)

	newObjDid, err := handler.currentObjectDid()
	require.Nil(t, err)
	require.NotEqual(t, objDid, newObjDid)

	// check that false in validator doesn't send prize
	playerTree, err = net.UpdateChainTree(playerTree, "jasons-game/inventory", map[string]interface{}{})
	require.Nil(t, err)

	validatorShouldPass = false
	err = handler.Handle(&jasonsgame.RequestObjectTransferMessage{
		From:   locTree.MustId(),
		To:     playerTree.MustId(),
		Object: newObjDid,
	})
	require.Nil(t, err)

	unchagedObjDid, err := handler.currentObjectDid()
	require.Nil(t, err)
	require.Equal(t, newObjDid, unchagedObjDid)
}
