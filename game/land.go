package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"

	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
)

type LandActor struct {
	middleware.LogAwareHolder
	network    network.Network
	did        string
	subscriber *actor.PID
}

type LandActorConfig struct {
	Network network.Network
	Did     string
}

func NewLandActorProps(cfg *LandActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &LandActor{
			did:     cfg.Did,
			network: cfg.Network,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (l *LandActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Started:
		l.subscriber = context.Spawn(l.network.PubSubSystem().NewSubscriberProps(topicFromDid(l.did)))
	case *TransferredObjectMessage:
		l.handleTransferredObject(context, msg)
	}
}

func (l *LandActor) handleTransferredObject(context actor.Context, msg *TransferredObjectMessage) {
	c := new(navigator.Cursor).SetChainTree(l.ChainTree()).SetLocation(msg.Loc[0], msg.Loc[1])
	loc, err := c.GetLocation()
	if err != nil {
		panic(fmt.Errorf("could not get location %v in did %v", msg.Loc, l.did))
	}

	obj := Object{Did: msg.Object}
	netobj := NetworkObject{Object: obj, Network: l.network}
	key, err := netobj.Name()

	if err != nil {
		panic(fmt.Errorf("error fetching object name for %v: %v", msg.Object, err))
	}

	if loc.Inventory == nil {
		loc.Inventory = make(map[string]string)
	}

	if _, ok := loc.Inventory[key]; ok {
		panic(fmt.Errorf("inventory for location %v already has key %s", loc.PrettyString(), key))
	}
	loc.Inventory[key] = obj.Did

	_, err = l.network.UpdateChainTree(l.ChainTree(), fmt.Sprintf("jasons-game/%d/%d", c.X(), c.Y()), loc)
	if err != nil {
		panic(fmt.Errorf("error updating chaintree: %v", err))
	}

	l.Log.Debugf("Object %v has been dropped at %v", obj.Did, loc.PrettyString())
}

func (l *LandActor) ChainTree() *consensus.SignedChainTree {
	tree, err := l.network.GetTree(l.did)
	if err != nil {
		panic(fmt.Errorf("could not find chaintree with did %v", l.did))
	}
	return tree
}
