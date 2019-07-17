package service

import (
	"context"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/community/pb/messages"
	"github.com/quorumcontrol/jasons-game/handlers/broadcast"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

func TestServiceActor(t *testing.T) {
	net := network.NewLocalNetwork()
	topic := []byte("some-topic")
	handler := broadcast.NewTopicBroadcastHandler(net, topic)

	actorCtx := actor.EmptyRootContext
	pid := actorCtx.Spawn(NewServiceActorProps(net, handler))
	chatMessage := &jasonsgame.ChatMessage{
		From:    "test",
		Message: "it works",
	}

	received := make(chan *jasonsgame.ChatMessage, 1)
	_, err := net.Community().Subscribe(topic, func(ctx context.Context, _ *messages.Envelope, msg proto.Message) {
		received <- msg.(*jasonsgame.ChatMessage)
	})
	require.Nil(t, err)

	serviceDid, err := actorCtx.RequestFuture(pid, &GetServiceDid{}, 5*time.Second).Result()
	require.Nil(t, err)

	err = net.Community().Send(net.Community().TopicFor(serviceDid.(string)), chatMessage)
	require.Nil(t, err)

	select {
	case msg := <-received:
		require.Equal(t, msg.From, chatMessage.From)
		require.Equal(t, msg.Message, chatMessage.Message)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timeout waiting for chat message")
	}
}
