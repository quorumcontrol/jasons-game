package autumn

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	messages "github.com/quorumcontrol/messages/build/go/community"
	"github.com/stretchr/testify/require"
)

func TestElementCombinerHandler(t *testing.T) {
	net := network.NewLocalNetwork()

	playerTree, err := net.CreateNamedChainTree("playerTree")
	require.Nil(t, err)

	serviceTree, err := net.CreateNamedChainTree("serviceTree")
	require.Nil(t, err)

	cfg := &autumnConfig{}
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

	client := &mockElementClient{
		net:     net,
		h:       h,
		player:  playerTree.MustId(),
		service: serviceTree.MustId(),
	}

	t.Run("with a correct combination", func(t *testing.T) {
		require.Nil(t, client.send(100))
		require.True(t, client.hasBowl())

		require.Nil(t, client.send(200))
		require.True(t, client.hasBowl())

		msg, err := client.pickupBowl()
		require.Nil(t, err)
		require.Empty(t, msg.Error)

		require.True(t, client.hasElement(300))
	})

	t.Run("with an incorrect combination", func(t *testing.T) {
		require.Nil(t, client.send(100))
		require.True(t, client.hasBowl())

		require.Nil(t, client.send(101))
		require.True(t, client.hasBowl())

		msg, _ := client.pickupBowl()
		require.NotNil(t, msg)
		require.Equal(t, msg.Error, fmt.Sprintf(combinationFailureMsg, "Weaver"))
		require.False(t, client.hasBowl())
	})

	t.Run("with a blocked combination", func(t *testing.T) {
		require.Nil(t, client.send(101))
		require.True(t, client.hasBowl())

		require.Nil(t, client.send(102))
		require.True(t, client.hasBowl())

		msg, _ := client.pickupBowl()
		require.NotNil(t, msg)
		require.Equal(t, msg.Error, combinationBlockedFailureMsg)
		require.False(t, client.hasBowl())
	})

	t.Run("with too few elements", func(t *testing.T) {
		require.Nil(t, client.send(100))
		require.True(t, client.hasBowl())

		msg, _ := client.pickupBowl()
		require.NotNil(t, msg)
		require.Equal(t, msg.Error, fmt.Sprintf(combinationNumFailureMsg, 2))
		require.True(t, client.hasBowl())
	})
}

type mockElementClient struct {
	net     network.Network
	h       *ElementCombinerHandler
	player  string
	service string
}

func (e *mockElementClient) send(id int) error {
	el, err := game.CreateObjectTree(e.net, (&element{ID: id}).Name())
	if err != nil {
		return err
	}
	msg := &jasonsgame.TransferredObjectMessage{
		From:   e.player,
		To:     e.service,
		Object: el.MustId(),
	}
	return e.h.Handle(msg)
}

func (e *mockElementClient) pickupBowl() (*jasonsgame.TransferredObjectMessage, error) {
	playerInventory, err := trees.FindInventoryTree(e.net, e.player)
	if err != nil {
		return nil, err
	}
	received := make(chan *jasonsgame.TransferredObjectMessage, 1)
	sub, err := e.net.Community().Subscribe(playerInventory.BroadcastTopic(), func(ctx context.Context, _ *messages.Envelope, msg proto.Message) {
		castMsg := msg.(*jasonsgame.TransferredObjectMessage)
		if castMsg != nil && castMsg.Object != "" && castMsg.Error == "" {
			_ = playerInventory.ForceAdd(castMsg.Object)
		}
		received <- castMsg
	})
	if err != nil {
		return nil, err
	}
	defer e.net.Community().Unsubscribe(sub) // nolint

	msg := &jasonsgame.RequestObjectTransferMessage{
		From:   e.service,
		To:     e.player,
		Object: e.bowl(),
	}

	_ = e.h.Handle(msg)

	select {
	case msg := <-received:
		return msg, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for transferred message")
	}
}

func (e *mockElementClient) bowl() string {
	serviceInventory, err := trees.FindInventoryTree(e.net, e.service)
	if err != nil {
		return ""
	}
	bowlDid, _ := serviceInventory.DidForName(combinationObjectName)
	return bowlDid
}

func (e *mockElementClient) hasBowl() bool {
	return len(e.bowl()) == 53
}

func (e *mockElementClient) hasElement(id int) bool {
	playerInventory, err := trees.FindInventoryTree(e.net, e.player)
	if err != nil {
		return false
	}

	elementDid, _ := playerInventory.DidForName((&element{ID: id}).Name())
	return len(elementDid) == 53
}
