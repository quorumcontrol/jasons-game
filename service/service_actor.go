package service

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

type ServiceActor struct {
	middleware.LogAwareHolder

	tree       *consensus.SignedChainTree
	network    network.Network
	subscriber *actor.PID
	handler    handlers.Handler
}

type GetServiceDid struct{}

func NewServiceActorProps(network network.Network, handler handlers.Handler) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &ServiceActor{
			network: network,
			handler: handler,
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
		s.Log.Infof("Received message %v", msg)
		err := s.handler.Handle(msg)
		if err != nil {
			s.Log.Errorf(fmt.Sprintf("error handling %v: %v", proto.MessageName(msg), err))
		}
	default:
		s.Log.Errorf("Unhandled message %v", msg)
	}
}

func (s *ServiceActor) initialize(actorCtx actor.Context) {
	var serviceTree *consensus.SignedChainTree
	var err error

	did := consensus.EcdsaPubkeyToDid(*s.network.PublicKey())

	serviceTree, err = s.network.GetTree(did)
	if err != nil {
		panic(fmt.Sprintf("err searching for tree %v\n", err))
	}
	if serviceTree == nil {
		serviceTree, err = consensus.NewSignedChainTree(*s.network.PublicKey(), s.network.TreeStore())
		if err != nil {
			panic(errors.Wrap(err, "error creating new signed chaintree"))
		}
	}

	serviceTree, err = s.network.UpdateChainTree(serviceTree, "jasons-game/handler/supports", s.handler.SupportedMessages())
	if err != nil {
		panic(errors.Wrap(err, "error updating handler/supports on service tree"))
	}

	s.tree = serviceTree
	topic := s.network.Community().TopicFor(s.tree.MustId())
	s.subscriber = actorCtx.Spawn(s.network.Community().NewSubscriberProps(topic))
}
