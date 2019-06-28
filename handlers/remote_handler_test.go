package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/quorumcontrol/community/pb/messages"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

func TestRemoteHandler(t *testing.T) {
	net := network.NewLocalNetwork()

	chatMessage := &jasonsgame.ChatMessage{
		From:    "test",
		Message: "it works",
	}

	handlerTree, err := net.CreateNamedChainTree("handlerTree")
	require.Nil(t, err)

	handlerTree, err = net.UpdateChainTree(handlerTree, "jasons-game/handler/supports", []string{proto.MessageName(chatMessage)})
	require.Nil(t, err)

	tree, err := net.CreateNamedChainTree("sourceTree")
	require.Nil(t, err)

	tree, err = net.UpdateChainTree(tree, HandlerPath, handlerTree.MustId())
	require.Nil(t, err)

	remoteHandler, err := FindHandlerForTree(net, tree.MustId())
	require.Nil(t, err)

	supports := remoteHandler.Supports(chatMessage)
	require.True(t, supports)

	require.Equal(t, remoteHandler.SupportedMessages(), []string{proto.MessageName(chatMessage)})

	received := make(chan *jasonsgame.ChatMessage, 1)
	_, err = net.Community().Subscribe(net.Community().TopicFor(handlerTree.MustId()), func(ctx context.Context, _ *messages.Envelope, msg proto.Message) {
		received <- msg.(*jasonsgame.ChatMessage)
	})
	require.Nil(t, err)

	err = remoteHandler.Handle(chatMessage)
	require.Nil(t, err)

	select {
	case msg := <-received:
		require.Equal(t, msg.From, chatMessage.From)
		require.Equal(t, msg.Message, chatMessage.Message)
	case <-time.After(1 * time.Second):
		require.Fail(t, "timeout waiting for chat message")
	}
}
