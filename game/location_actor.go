package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

// jasonsgame/interactions/go/north

type LocationActor struct {
	middleware.LogAwareHolder
	did            string
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

type Interaction struct {
	Command string
	Action  string
	Args    map[string]string
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
		_, err := l.network.Community().SubscribeActor(context.Self(), topicFor(l.did))
		if err != nil {
			panic(errors.Wrap(err, "error spawning land actor subscription"))
		}
		l.inventoryActor = actorCtx.Spawn(NewInventoryActorProps(&InventoryActorConfig{
			Did:     l.did,
			Network: l.network,
		}))
	case *GetLocation:
		actorCtx.Respond(&jasonsgame.Location{})
	case *GetInteraction:
		resp, _, err := l.SignedTree().ChainTree.Dag.Resolve(append([]string{"tree", "data", "jasons-game", "interactions", msg.Command}))
		if err != nil {
			panic(err)
		}

		var interaction Interaction

		err = mapstructure.Decode(resp, &interaction)
		if err != nil {
			panic(err)
		}

		interaction.Command = msg.Command

		actorCtx.Respond(&interaction)
	case *InventoryListRequest:
		actorCtx.Forward(l.inventoryActor)
	case *TransferObjectRequest:
		actorCtx.Forward(l.inventoryActor)
	}
}

func (l *LocationActor) SignedTree() *consensus.SignedChainTree {
	tree, err := l.network.GetTree(l.did)
	if err != nil {
		panic(fmt.Errorf("could not find chaintree with did %v", l.did))
	}
	return tree
}
