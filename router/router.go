package router

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/messages"
	"github.com/quorumcontrol/jasons-game/network"
	gossip3messages "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

type Router struct {
	middleware.LogAwareHolder

	network       network.Network
	msgSubscriber *actor.PID
	uiActor       *actor.PID
	gameActor     *actor.PID
	playerId      string
}

func NewRouterProps(network network.Network, uiActor *actor.PID) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		r := &Router{
			network: network,
			uiActor: uiActor,
		}
		return r
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (r *Router) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		if err := r.initialize(actorCtx); err != nil {
			r.Log.Warnw("failure to initialize", "error", err)
		}
	case messages.PlayerMessage:
		if err := r.routeMessage(actorCtx, msg); err != nil {
			r.Log.Warnw("failure routing message", "error", err)
		}
	case gossip3messages.WireMessage:
		r.Log.Warnw("received message of unrecognized type", "typeCode", msg.TypeCode())
	}
}

func (r *Router) routeMessage(actorCtx actor.Context, msg messages.PlayerMessage) error {
	if msg.ToPlayer() != r.playerId {
		r.Log.Debugw("ignoring game message as it's not for us", "from", msg.FromPlayer(),
			"to", msg.ToPlayer(), "ourId", r.playerId)
		return nil
	}

	r.Log.Debugw("handling game message", "from", msg.FromPlayer(), "to", msg.ToPlayer())
	switch m := msg.(type) {
	case *messages.OpenPortalMessage:
		r.Log.Debugw("received OpenPortalMessage, forwarding to game actor", "msg", m)
		actorCtx.Forward(r.gameActor)
	case *messages.OpenPortalResponseMessage:
		r.Log.Debugw("received OpenPortalResponseMessage, forwarding to game actor", "msg", m)
		actorCtx.Forward(r.gameActor)
	}

	return nil
}

func (r *Router) initialize(actorCtx actor.Context) error {
	r.Log.Debugw("initializing")

	playerTree, err := game.GetPlayerTree(r.network)
	if err != nil {
		return err
	}

	playerId := playerTree.Did()
	r.playerId = playerId

	r.msgSubscriber = actorCtx.Spawn(r.network.PubSubSystem().NewSubscriberProps(
		network.GeneralTopic))
	r.Log.Debugw("subscribed to general pubsub topic", "topic", network.GeneralTopic)

	broadcaster := messages.NewBroadcaster(r.network)
	r.gameActor = actor.EmptyRootContext.Spawn(game.NewGameProps(playerTree, r.uiActor,
		r.network, broadcaster))
	return nil
}
