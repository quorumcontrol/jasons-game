package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

// jasonsgame/interactions/go/north

type LocationActor struct {
	middleware.LogAwareHolder
	did            string
	location       *LocationTree
	network        network.Network
	inventoryActor *actor.PID
}

type LocationActorConfig struct {
	Network network.Network
	Did     string
}

type GetLocation struct{}

type GetInteraction struct {
	Command string
}

type SetLocationDescriptionRequest struct {
	Description string
}

type SetLocationDescriptionResponse struct {
	Error error
}

func NewLocationActorProps(cfg *LocationActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &LocationActor{
			did:     cfg.Did,
			network: cfg.Network,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (l *LocationActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		tree, err := l.network.GetTree(l.did)
		if err != nil {
			panic("could not find location")
		}
		l.location = NewLocationTree(l.network, tree)

		_, err = l.network.Community().SubscribeActor(actorCtx.Self(), topicFor(l.did))
		if err != nil {
			panic(errors.Wrap(err, "error spawning land actor subscription"))
		}

		l.inventoryActor = actorCtx.Spawn(NewInventoryActorProps(&InventoryActorConfig{
			Did:     l.did,
			Network: l.network,
		}))
	case *GetLocation:
		desc, err := l.location.GetDescription()
		if err != nil {
			panic(fmt.Errorf("error getting description: %v", err))
		}

		actorCtx.Respond(&jasonsgame.Location{
			Did:         l.location.MustId(),
			Tip:         l.location.Tip().String(),
			Description: desc,
		})
	case *GetInteraction:
		interaction, err := l.location.GetInteraction(msg.Command)
		if err != nil {
			panic(errors.Wrap(err, "error fetching interaction"))
		}
		actorCtx.Respond(interaction)
	case *SetLocationDescriptionRequest:
		err := l.location.SetDescription(msg.Description)
		actorCtx.Respond(&SetLocationDescriptionResponse{Error: err})
	case *InventoryListRequest:
		actorCtx.Forward(l.inventoryActor)
	case *TransferObjectRequest:
		actorCtx.Forward(l.inventoryActor)
	}
}

func (l *LocationActor) MustResolve(path []string) interface{} {
	resp, _, err := l.SignedTree().ChainTree.Dag.Resolve(path)
	if err != nil {
		panic(fmt.Errorf("could not find chaintree: %v", err))
	}
	return resp
}

func (l *LocationActor) SignedTree() *consensus.SignedChainTree {
	tree, err := l.network.GetTree(l.did)
	if err != nil {
		panic(fmt.Errorf("could not find chaintree with did %v", l.did))
	}
	return tree
}
