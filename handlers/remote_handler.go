package handlers

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/network"
)

type RemoteHandler struct {
	Handler
	did               string
	net               network.Network
	supportedMessages HandlerMessageList
}

func (h *RemoteHandler) Handle(msg proto.Message) error {
	if !h.Supports(msg) {
		return ErrUnsupportedMessageType
	}
	topic := h.net.Community().TopicFor(h.did)
	return h.net.Community().Send(topic, msg)
}

func (h *RemoteHandler) Supports(msg proto.Message) bool {
	return h.supportedMessages.Contains(msg)
}

func (h *RemoteHandler) SupportedMessages() []string {
	return h.supportedMessages
}

func (h *RemoteHandler) Did() string {
	return h.did
}

func FindHandlerForTree(net network.Network, did string) (*RemoteHandler, error) {
	ctx := context.TODO()

	tree, err := net.GetTree(did)
	if err != nil {
		return nil, err
	}

	handlerDid, _, err := tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", HandlerPath})
	if err != nil {
		return nil, err
	}

	if handlerDid == nil {
		return nil, nil
	}

	handlerTree, err := net.GetTree(handlerDid.(string))
	if err != nil {
		return nil, err
	}

	supports, _, err := handlerTree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "handler", "supports"})
	if err != nil {
		return nil, err
	}
	if supports == nil {
		supports = []interface{}{}
	}

	supportedMessages := make(HandlerMessageList, len(supports.([]interface{})))
	for i, typeString := range supports.([]interface{}) {
		supportedMessages[i] = typeString.(string)
	}

	return &RemoteHandler{
		did:               handlerDid.(string),
		net:               net,
		supportedMessages: supportedMessages,
	}, nil
}
