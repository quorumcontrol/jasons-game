package court

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

func TestPrizeHandler(t *testing.T) {
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

	prize, err := handler.resolvePrize()
	require.Nil(t, err)
	require.Equal(t, uint64(1), prize.Count)

	firstWinnerPlayer, _, err := handler.Tree().ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "winners", "100", "1", "player", "id"})
	require.Nil(t, err)
	require.Equal(t, firstWinnerPlayer, playerTree.MustId())

	firstWinnerPrize, _, err := handler.Tree().ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "winners", "100", "1", "prize", "id"})
	require.Nil(t, err)
	require.Equal(t, firstWinnerPrize, objDid)

	prizeObj, err := game.FindObjectTree(net, objDid)
	require.Nil(t, err)

	prizeName, err := prizeObj.GetName()
	require.Equal(t, prizeName, "test-prize")
	require.Nil(t, err)

	objDid, err = waitForNewObj(handler, objDid)
	require.Nil(t, err)

	// fake lots of winners
	handler.tree, err = net.UpdateChainTree(handler.Tree(), prizePath, Prize{Count: 344})
	require.Nil(t, err)

	player2Tree, err := net.CreateChainTree()
	require.Nil(t, err)

	err = handler.Handle(&jasonsgame.RequestObjectTransferMessage{
		From:   locTree.MustId(),
		To:     player2Tree.MustId(),
		Object: objDid,
	})
	require.Nil(t, err)

	prize, err = handler.resolvePrize()
	require.Nil(t, err)
	require.Equal(t, uint64(345), prize.Count)

	firstWinnerPlayer, _, err = handler.Tree().ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "winners", "100", "1", "player", "id"})
	require.Nil(t, err)
	require.Equal(t, firstWinnerPlayer, playerTree.MustId())

	otherWinnerPlayer, _, err := handler.Tree().ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "winners", "400", "345", "player", "id"})
	require.Nil(t, err)
	require.Equal(t, otherWinnerPlayer, player2Tree.MustId())

	otherWinnerPrize, _, err := handler.Tree().ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "winners", "400", "345", "prize", "id"})
	require.Nil(t, err)
	require.Equal(t, otherWinnerPrize, objDid)

	objDid, err = waitForNewObj(handler, objDid)
	require.Nil(t, err)

	// check that false in validator doesn't send prize
	playerTree, err = net.CreateChainTree()
	require.Nil(t, err)

	validatorShouldPass = false
	err = handler.Handle(&jasonsgame.RequestObjectTransferMessage{
		From:   locTree.MustId(),
		To:     playerTree.MustId(),
		Object: objDid,
	})
	require.Nil(t, err)

	time.Sleep(50 * time.Millisecond)

	unchagedObjDid, err := handler.currentObjectDid()
	require.Nil(t, err)
	require.Equal(t, objDid, unchagedObjDid)
}

func waitForNewObj(handler *PrizeHandler, currentObjDid string) (newObjDid string, err error) {
	newObjDidCh := make(chan string, 1)
	go func() {
		for i := 0; i < 20; i++ {
			newDid, _ := handler.currentObjectDid()
			if newDid != "" && newDid != currentObjDid {
				newObjDidCh <- newDid
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	select {
	case did := <-newObjDidCh:
		newObjDid = did
	case <-time.After(3 * time.Second):
		return "", fmt.Errorf("timeout waiting for new object to spawn")
	}
	return newObjDid, nil
}
