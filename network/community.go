package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/golang/protobuf/proto"
	"github.com/quorumcontrol/community/client"
	"github.com/quorumcontrol/community/config"
	"github.com/quorumcontrol/community/pb/messages"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/quorumcontrol/tupelo-go-sdk/tracing"
)

type Community struct {
	client *client.Client
}

const CommunityName = "jasonsgame"

func NewJasonCommunity(ctx context.Context, key *ecdsa.PrivateKey, p2pHost *p2p.LibP2PHost) *Community {
	clientConfig := &config.ClientConfig{
		Name:            CommunityName,
		Shards:          1024,
		LocalIdentifier: []byte(p2pHost.Identity()),
		PubSub:          p2pHost.GetPubSub(),
		Key:             key,
	}

	return &Community{
		client: client.New(ctx, clientConfig),
	}
}

func (c *Community) Send(topicOrTo []byte, msg proto.Message) error {
	env := &messages.Envelope{To: topicOrTo}
	return c.client.SendProtobuf(env, msg)
}

func (c *Community) Unsubscribe(subscription *client.Subscription) error {
	return c.client.Unsubscribe(subscription)
}

func (c *Community) Subscribe(topicOrTo []byte, fn func(ctx context.Context, env *messages.Envelope, msg proto.Message)) (*client.Subscription, error) {
	return c.client.SubscribeProtobuf(topicOrTo, fn)
}

func (c *Community) SubscribeActor(pid *actor.PID, topicOrTo []byte) (*client.Subscription, error) {
	return c.Subscribe(topicOrTo, func(ctx context.Context, _ *messages.Envelope, msg proto.Message) {
		actor.EmptyRootContext.Send(pid, msg)
	})
}

type communitySubscriberActor struct {
	middleware.LogAwareHolder
	tracing.ContextHolder

	topicOrTo    []byte
	community    *Community
	subscription *client.Subscription
}

func (cs *communitySubscriberActor) Receive(actorContext actor.Context) {
	switch msg := actorContext.Message().(type) {
	case *actor.Started:
		cs.Log.Debugw("subscribed", "topic", cs.topicOrTo)
		sub, err := cs.community.SubscribeActor(actorContext.Self(), cs.topicOrTo)
		if err != nil {
			panic(fmt.Sprintf("subscription failed, dying %v", err))
		}
		cs.subscription = sub
	case *actor.Stopping:
		err := cs.community.Unsubscribe(cs.subscription)
		if err != nil {
			cs.Log.Errorw("error unsubscribing", "err", err)
		}
	case proto.Message:
		actorContext.Send(actorContext.Parent(), msg)
	}
}

func (c *Community) NewSubscriberProps(topicOrTo []byte) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &communitySubscriberActor{
			topicOrTo: topicOrTo,
			community: c,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}
