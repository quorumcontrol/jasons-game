package service

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/AsynkronIT/protoactor-go/router"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

type ServiceActor struct {
	middleware.LogAwareHolder

	tree               *consensus.SignedChainTree
	network            network.Network
	subscriber         *actor.PID
	handler            handlers.Handler
	handlerPool        *actor.PID
	handlerConcurrency int
}

type ServiceActorConfig struct {
	Network     network.Network
	Handler     handlers.Handler
	Tree        *consensus.SignedChainTree
	Concurrency int
}

type GetServiceDid struct{}

func NewServiceActorProps(network network.Network, handler handlers.Handler) *actor.Props {
	return NewServiceActorPropsWithConfig(&ServiceActorConfig{
		Network: network,
		Handler: handler,
	})
}

func NewServiceActorPropsWithTree(network network.Network, handler handlers.Handler, tree *consensus.SignedChainTree) *actor.Props {
	return NewServiceActorPropsWithConfig(&ServiceActorConfig{
		Network: network,
		Handler: handler,
		Tree:    tree,
	})
}

func NewServiceActorPropsWithConfig(cfg *ServiceActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &ServiceActor{
			network:            cfg.Network,
			handler:            cfg.Handler,
			tree:               cfg.Tree,
			handlerConcurrency: cfg.Concurrency,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (s *ServiceActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		s.initialize(actorCtx)
	case *GetServiceDid:
		actorCtx.Respond(s.tree.MustId())
	case proto.Message:
		s.Log.Debugf("Received message %v", msg)
		actorCtx.Forward(s.handlerPool)
	default:
		s.Log.Errorf("Unhandled message %v", msg)
	}
}

func (s *ServiceActor) initialize(actorCtx actor.Context) {
	var err error

	if s.handlerConcurrency <= 0 {
		s.handlerConcurrency = 1
	}

	if s.tree == nil {
		did := consensus.EcdsaPubkeyToDid(*s.network.PublicKey())

		s.tree, err = s.network.GetTree(did)
		if err != nil {
			panic(fmt.Sprintf("err searching for tree %v\n", err))
		}
		if s.tree == nil {
			s.tree, err = consensus.NewSignedChainTree(*s.network.PublicKey(), s.network.TreeStore())
			if err != nil {
				panic(errors.Wrap(err, "error creating new signed chaintree"))
			}
		}
	}

	s.tree, err = s.network.UpdateChainTree(s.tree, "jasons-game/handler/supports", s.handler.SupportedMessages())
	if err != nil {
		panic(errors.Wrap(err, "error updating handler/supports on service tree"))
	}

	topic := s.network.Community().TopicFor(s.tree.MustId())

	s.handlerPool = actorCtx.Spawn(router.NewRoundRobinPool(s.handlerConcurrency).WithFunc(func(poolCtx actor.Context) {
		switch msg := poolCtx.Message().(type) {
		case proto.Message:
			err := s.handler.Handle(msg)
			if err != nil {
				s.Log.Error(err)
			}
		}
	}))

	s.subscriber = actorCtx.Spawn(s.network.Community().NewSubscriberProps(topic))
}
