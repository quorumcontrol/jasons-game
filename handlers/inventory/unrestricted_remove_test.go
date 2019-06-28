package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/quorumcontrol/community/pb/messages"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

func TestUnrestrictedRemoveHandler(t *testing.T) {
	net := network.NewLocalNetwork()

	fromTree, err := net.CreateNamedChainTree("fromTree")
	require.Nil(t, err)

	toTree, err := net.CreateNamedChainTree("toTree")
	require.Nil(t, err)

	objectTree, err := net.CreateNamedChainTree("objectTree")
	require.Nil(t, err)
	objectTree, err = net.UpdateChainTree(objectTree, "jasons-game/name", "some obj")
	require.Nil(t, err)

	fromInventory, err := trees.FindInventoryTree(net, fromTree.MustId())
	require.Nil(t, err)

	err = fromInventory.Add(objectTree.MustId())
	require.Nil(t, err)

	existsBeforeInFrom, err := fromInventory.Exists(objectTree.MustId())
	require.Nil(t, err)
	require.True(t, existsBeforeInFrom)

	toInventory, err := trees.FindInventoryTree(net, toTree.MustId())
	require.Nil(t, err)
	existsBeforeInTo, err := toInventory.Exists(objectTree.MustId())
	require.Nil(t, err)
	require.False(t, existsBeforeInTo)

	msg := &jasonsgame.RequestTransferObjectMessage{
		From:   fromTree.MustId(),
		To:     toTree.MustId(),
		Object: objectTree.MustId(),
	}

	h := NewUnrestrictedRemoveHandler(net)
	require.True(t, h.Supports(msg))

	t.Run("without handler on target inventory", func(t *testing.T) {
		received := make(chan *jasonsgame.TransferredObjectMessage, 1)
		net.Community().Subscribe(toInventory.BroadcastTopic(), func(ctx context.Context, _ *messages.Envelope, msg proto.Message) {
			received <- msg.(*jasonsgame.TransferredObjectMessage)
		})

		err = h.Handle(msg)
		require.Nil(t, err)

		select {
		case receivedMsg := <-received:
			require.Equal(t, receivedMsg.From, msg.From)
			require.Equal(t, receivedMsg.To, msg.To)
			require.Equal(t, receivedMsg.Object, msg.Object)
		case <-time.After(2 * time.Second):
			require.Fail(t, "timeout waiting for transferred message")
		}
	})

	t.Run("with handler on target inventory", func(t *testing.T) {
		handlerTree, err := net.CreateNamedChainTree("handlerTree")
		require.Nil(t, err)

		handlerTree, err = net.UpdateChainTree(handlerTree, "jasons-game/handler/supports", []string{proto.MessageName((*jasonsgame.TransferredObjectMessage)(nil))})
		require.Nil(t, err)

		toTree, err = net.UpdateChainTree(toTree, "jasons-game-handler", handlerTree.MustId())
		require.Nil(t, err)

		received := make(chan *jasonsgame.TransferredObjectMessage, 1)
		net.Community().Subscribe(net.Community().TopicFor(handlerTree.MustId()), func(ctx context.Context, _ *messages.Envelope, msg proto.Message) {
			received <- msg.(*jasonsgame.TransferredObjectMessage)
		})

		err = h.Handle(msg)
		require.Nil(t, err)

		select {
		case receivedMsg := <-received:
			require.Equal(t, receivedMsg.From, msg.From)
			require.Equal(t, receivedMsg.To, msg.To)
			require.Equal(t, receivedMsg.Object, msg.Object)
		case <-time.After(2 * time.Second):
			require.Fail(t, "timeout waiting for transferred message")
		}
	})
}