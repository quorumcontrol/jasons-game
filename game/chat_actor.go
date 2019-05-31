package game

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

const chatSuffix = "/chat"

func chatTopicFromDid(did string) []byte {
	return topicFor(did + chatSuffix)
}

type ChatActor struct {
	middleware.LogAwareHolder
	network network.Network
	did     string
}

type ChatActorConfig struct {
	Network network.Network
	Did     string
}

func NewChatActorProps(cfg *ChatActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &ChatActor{
			network: cfg.Network,
			did:     cfg.Did,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (c *ChatActor) Receive(actorCtx actor.Context) {
	switch actorCtx.Message().(type) {
	case *actor.Started:
		c.network.Community().SubscribeActor(actorCtx.Self(), chatTopicFromDid(c.did))
	}
}
