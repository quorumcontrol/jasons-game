package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"

	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
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

	if loc.Did == msg.To {
		l.handleIncomingObject(context, loc, msg)
	} else if loc.Did == msg.From {
		l.handleOutgoingObject(context, loc, msg)
	} else {
		panic(fmt.Errorf("transferred object %v does not refer to this location %v", msg, l.did))
	}
}

func (l *LandActor) handleIncomingObject(context actor.Context, loc *jasonsgame.Location, msg *TransferredObjectMessage) {
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

	_, err = l.network.UpdateChainTree(l.ChainTree(), fmt.Sprintf("jasons-game/%d/%d", loc.X, loc.Y), loc)
	if err != nil {
		panic(fmt.Errorf("error updating chaintree: %v", err))
	}

	l.Log.Debugf("Object %v has been dropped at %v", obj.Did, loc.PrettyString())
	fmt.Printf("Object %v has been dropped at %v", obj.Did, loc.PrettyString())

}

func (l *LandActor) handleOutgoingObject(context actor.Context, loc *jasonsgame.Location, msg *TransferredObjectMessage) {
	obj := Object{Did: msg.Object}
	netobj := NetworkObject{Object: obj, Network: l.network}
	key, err := netobj.Name()

	if err != nil {
		panic(fmt.Errorf("error fetching object name for %v: %v", msg.Object, err))
	}

	if loc.Inventory == nil {
		loc.Inventory = make(map[string]string)
	}

	if _, ok := loc.Inventory[key]; !ok {
		panic(fmt.Errorf("inventory for location %v does not have key %s", loc.PrettyString(), key))
	}

	playerChainTree, err := l.network.GetTree(msg.To)
	if err != nil {
		panic(fmt.Errorf("error fetching player chaintree %v: %v", msg.To, err))
	}

	playerTree := NewPlayerTree(l.network, playerChainTree)

	playerKeys, err := playerTree.Keys()
	if err != nil {
		panic(fmt.Errorf("error fetching player keys %v: %v", msg.To, err))
	}

	objChaintree, err := netobj.ChainTree()
	if err != nil {
		panic(fmt.Errorf("error fetching object chaintree %v: %v", obj.Did, err))
	}

	_, err = l.network.ChangeChainTreeOwner(objChaintree, playerKeys)
	if err != nil {
		panic(fmt.Errorf("error changing owner for object %s; error: %v", obj.Did, err))
	}

	delete(loc.Inventory, key)

	_, err = l.network.UpdateChainTree(l.ChainTree(), fmt.Sprintf("jasons-game/%d/%d", loc.X, loc.Y), loc)
	if err != nil {
		panic(fmt.Errorf("error updating chaintree: %v", err))
	}

	l.Log.Debugf("Object %v has been picked up from %v by %v", obj.Did, loc.PrettyString(), playerTree.Did())
}

func (l *LandActor) ChainTree() *consensus.SignedChainTree {
	tree, err := l.network.GetTree(l.did)
	if err != nil {
		panic(fmt.Errorf("could not find chaintree with did %v", l.did))
	}
	return tree
}
