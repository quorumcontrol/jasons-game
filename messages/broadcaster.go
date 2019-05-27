package messages

import (
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
)

type Broadcaster struct {
	network network.Network
}

func NewBroadcaster(network network.Network) *Broadcaster {
	return &Broadcaster{
		network: network,
	}
}

func (b *Broadcaster) BroadcastGeneral(msg messages.WireMessage) error {
	return b.Broadcast(network.GeneralTopic, msg)
}

func (b *Broadcaster) Broadcast(topic string, msg messages.WireMessage) error {
	return b.network.PubSubSystem().Broadcast(topic, msg)
}
