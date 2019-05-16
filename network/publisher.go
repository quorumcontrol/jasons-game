package network

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
)

func newPublisherProps(pubsubSystem remote.PubSub) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &publisher{
			pubsubSystem: pubsubSystem,
		}
	})
}

type publisher struct {
	pubsubSystem remote.PubSub
}

func (p *publisher) Receive(aCtx actor.Context) {
	switch msg := aCtx.Message().(type) {
	case *cbornode.Node:
		p.pubsubSystem.Broadcast(BlockTopic, &Block{Cid: msg.Cid().Bytes()})
	}
}
