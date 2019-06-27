package game

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

const chatTopicSuffix = "/chat"

type ChatActor struct {
	middleware.LogAwareHolder
	did        string
	community  *network.Community
	subscriber *actor.PID
}

type ChatActorConfig struct {
	Did       string
	Community *network.Community
}

func NewChatActorProps(cfg *ChatActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &ChatActor{
			did:       cfg.Did,
			community: cfg.Community,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (c *ChatActor) chatTopic() []byte {
	return c.community.TopicFor(c.did)
}

func (c *ChatActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		c.subscriber = actorCtx.Spawn(c.community.NewSubscriberProps(c.chatTopic()))
	case string:
		err := c.community.Send(c.chatTopic(), &jasonsgame.ChatMessage{Message: msg})
		if err != nil {
			panic(errors.Wrap(err, "failed to broadcast ChatMessage"))
		}
	case *jasonsgame.ChatMessage:
		actorCtx.Send(actorCtx.Parent(), msg)
	}
}
