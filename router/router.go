package router

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/messages"
	gossip3messages "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
	"go.uber.org/zap"
)

type Router struct {
	network       network.Network
	log           *zap.SugaredLogger
	msgSubscriber *actor.PID
	uiActor *actor.PID
	gameActor *actor.PID
}

func NewRouterProps(network network.Network, uiActor *actor.PID) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Router{
			network: network,
			uiActor: uiActor,
			log:     middleware.Log.Named("router"),
		}
	})
}

func (r *Router) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		r.initialize(actorCtx)
	case *messages.OpenPortalMessage:
		r.log.Debugw("received OpenPortalMessage", "msg", msg)
		// TODO: Route to correct actor
	case gossip3messages.WireMessage:
		r.log.Warnw("received message of unrecognized type", "typeCode", msg.TypeCode())
	}
}

func (r *Router) initialize(actorCtx actor.Context) {
	r.log.Debugw("initializing")
	r.msgSubscriber = actorCtx.Spawn(r.network.PubSubSystem().NewSubscriberProps(
		network.GeneralTopic))
	r.log.Debugw("subscribed to general pubsub topic", "topic", network.GeneralTopic)

	broadcaster := messages.NewBroadcaster(r.network)
	r.gameActor = actor.EmptyRootContext.Spawn(game.NewGameProps(r.uiActor, r.network, broadcaster))
}
