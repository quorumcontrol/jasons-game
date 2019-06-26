package broadcast

import (
	"github.com/golang/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
)

type TopicBroadcastHandler struct {
	network network.Network
	topic   []byte
}

func NewTopicBroadcastHandler(network network.Network, topic []byte) handlers.Handler {
	return &TopicBroadcastHandler{
		network: network,
		topic:   topic,
	}
}

func (h *TopicBroadcastHandler) Handle(msg proto.Message) error {
	return h.network.Community().Send(h.topic, msg)
}

func (h *TopicBroadcastHandler) Supports(msg proto.Message) bool {
	return true
}

func (h *TopicBroadcastHandler) SupportedMessages() []string {
	return []string{}
}
