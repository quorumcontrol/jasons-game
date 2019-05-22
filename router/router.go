package router

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/jasons-game/messages"
	"github.com/quorumcontrol/jasons-game/network"
	gossip3messages "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

type Router struct {
	middleware.LogAwareHolder

	network       network.Network
	msgSubscriber *actor.PID
	table         map[string]*actor.PID
}

func NewRouterProps(network network.Network, table map[string]*actor.PID) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		r := &Router{
			network: network,
			table:   table,
		}
		return r
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

type AddRouteMessage struct {
	Did string
	Pid *actor.PID
}

func (r *Router) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		if err := r.initialize(actorCtx); err != nil {
			r.Log.Warnw("failure to initialize", "error", err)
		}
	case AddRouteMessage:
		if err := r.addRoute(actorCtx, msg); err != nil {
			panic(err)
		}
	case messages.PlayerMessage:
		if err := r.routeMessage(actorCtx, msg); err != nil {
			r.Log.Warnw("failure routing message", "error", err)
		}
	case gossip3messages.WireMessage:
		r.Log.Warnw("received message of unrecognized type", "typeCode", msg.TypeCode())
	}
}

func (r *Router) addRoute(actorCtx actor.Context, msg AddRouteMessage) error {
	r.Log.Debugw("adding route", "DID", msg.Did, "PID", msg.Pid)
	if _, ok := r.table[msg.Did]; ok {
		panic(fmt.Sprintf("DID %s already has a route", msg.Did))
	}

	r.table[msg.Did] = msg.Pid
	return nil
}

func (r *Router) routeMessage(actorCtx actor.Context, msg messages.PlayerMessage) error {
	r.Log.Debugw("received message")
	act, ok := r.table[msg.ToDid()]
	if !ok {
		r.Log.Debugw("the message doesn't address any of our objects, ignoring")
		return nil
	}

	r.Log.Debugw("passing game message to correct actor", "from", msg.FromPlayer())
	actorCtx.Forward(act)

	return nil
}

func (r *Router) initialize(actorCtx actor.Context) error {
	r.Log.Debugw("initializing")

	r.msgSubscriber = actorCtx.Spawn(r.network.PubSubSystem().NewSubscriberProps(
		network.GeneralTopic))
	r.Log.Debugw("subscribed to general pubsub topic", "topic", network.GeneralTopic)

	return nil
}
