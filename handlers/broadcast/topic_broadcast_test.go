package broadcast

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/community/pb/messages"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

func TestTopicBroadcastHandler(t *testing.T) {
	net := network.NewLocalNetwork()
	topic := []byte("some-topic")
	h := NewTopicBroadcastHandler(net, topic)

	chatMessage := &jasonsgame.ChatMessage{
		From:    "test",
		Message: "it works",
	}

	supports := h.Supports(chatMessage)
	require.True(t, supports)

	received := make(chan *jasonsgame.ChatMessage, 1)
	_, err := net.Community().Subscribe(topic, func(ctx context.Context, _ *messages.Envelope, msg proto.Message) {
		received <- msg.(*jasonsgame.ChatMessage)
	})
	require.Nil(t, err)
	err = h.Handle(chatMessage)
	require.Nil(t, err)

	select {
	case msg := <-received:
		require.Equal(t, msg.From, chatMessage.From)
		require.Equal(t, msg.Message, chatMessage.Message)
	case <-time.After(1 * time.Second):
		require.Fail(t, "timeout waiting for chat message")
	}
}
