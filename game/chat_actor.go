package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

const chatSuffix = "/chat"

func chatTopicFromDid(did string) []byte {
	return topicFor(did + chatSuffix)
}

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

func (c *ChatActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		c.subscriber = actorCtx.Spawn(c.community.NewSubscriberProps(chatTopicFromDid(c.did)))
	case string:
		err := c.community.Send(chatTopicFromDid(c.did), &jasonsgame.ChatMessage{Message: msg})
		if err != nil {
			panic(fmt.Errorf("failed to broadcast ChatMessage: %s", err))
		}
	case *jasonsgame.ChatMessage:
		actorCtx.Send(actorCtx.Parent(), msg)
	}
}
