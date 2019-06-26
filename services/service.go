package services

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var log = logging.Logger("service")

func init() {
	logging.SetLogLevel("service", "info")
}

type Service struct {
	tree       *consensus.SignedChainTree
	network    network.Network
	subscriber *actor.PID
	handler    handlers.Handler
}

func NewServiceProps(network network.Network, handler handlers.Handler) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Service{
			network: network,
			handler: handler,
		}
	})
}

func (s *Service) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		s.initialize(actorCtx)
	case proto.Message:
		log.Infof("Received message %v", msg)
		err := s.handler.Handle(msg)
		if err != nil {
			log.Errorf(fmt.Sprintf("error handling %v: %v", proto.MessageName(msg), err))
		}
	default:
		log.Errorf("Unhandled message %v", msg)
	}
}

func (s *Service) initialize(actorCtx actor.Context) {
	serviceTree, err := s.network.GetChainTreeByName("serviceTree")
	if err != nil {
		panic(errors.Wrap(err, "fetching service chain"))
	}

	if serviceTree == nil {
		serviceTree, err = s.network.CreateNamedChainTree("serviceTree")

		if err != nil {
			panic(errors.Wrap(err, "creating service chain"))
		}
	}

	serviceTree, err = s.network.UpdateChainTree(serviceTree, "jasons-game/supports", s.handler.SupportedMessages())
	if err != nil {
		panic(err)
	}

	s.tree = serviceTree

	topic := s.network.Community().TopicFor(s.tree.MustId())
	s.subscriber = actorCtx.Spawn(s.network.Community().NewSubscriberProps(topic))

	log.Infof("Starting listener with ChainTree id %v", s.tree.MustId())
}
