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
		nodeCid := msg.Cid()
		log.Debugf("publishing block %s", nodeCid.String())
		if err := p.pubsubSystem.Broadcast(
			BlockTopic,
			&Block{
				Cid:  nodeCid.Bytes(),
				Data: msg.RawData(),
			},
		); err != nil {
			log.Errorf("failed to broadcast: %s", err)
		}
	}
}
