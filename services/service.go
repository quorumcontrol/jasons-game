package services

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var log = logging.Logger("service")

func init() {
	logging.SetLogLevel("service", "info")
}

type Service struct {
	tree            *consensus.SignedChainTree
	network         network.Network
	subscriber      *actor.PID
	handlerRegistry *HandlerRegistry
}

type AttachHandler struct {
	HandlerProps *actor.Props
}

func NewServiceProps(network network.Network) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Service{
			network: network,
		}
	})
}

func (s *Service) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		s.initialize(actorCtx)
	case *AttachHandler:
		pid := actorCtx.Spawn(msg.HandlerProps)
		s.handlerRegistry.Add(pid)

		newTree, err := s.network.UpdateChainTree(s.tree, "jasons-game/supports", s.handlerRegistry.AllMessages())
		if err != nil {
			panic(err)
		}
		s.tree = newTree
	case proto.Message:
		handlerActors := s.handlerRegistry.ForMessage(proto.MessageName(msg))

		if len(handlerActors) == 0 {
			log.Errorf("Unhandled message %v", msg)
			return
		}

		for _, pid := range handlerActors {
			actorCtx.Forward(pid)
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

	s.tree = serviceTree
	s.subscriber = actorCtx.Spawn(s.network.Community().NewSubscriberProps([]byte(s.tree.MustId())))

	s.handlerRegistry = NewHandlerRegistry()
	log.Infof("Starting listener with ChainTree id %v", s.tree.MustId())
}
