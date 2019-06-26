package handlers

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/network"
)

type RemoteHandler struct {
	Handler
	did               string
	net               network.Network
	supportedMessages map[string]bool
}

func (h *RemoteHandler) Send(msg proto.Message) error {
	if !h.Supports(msg) {
		return fmt.Errorf("handler does not support message %v", msg)
	}

	topic := h.net.Community().TopicFor(h.did)
	return h.net.Community().Send(topic, msg)
}

func (h *RemoteHandler) Supports(msg proto.Message) bool {
	msgType := proto.MessageName(msg)
	return h.SupportsType(msgType)
}

func (h *RemoteHandler) SupportsType(msgType string) bool {
	return h.supportedMessages[msgType] || false
}

func FindHandlerForTree(net network.Network, did string) (*RemoteHandler, error) {
	tree, err := net.GetTree(did)
	if err != nil {
		return nil, err
	}

	handlerDid, _, err := tree.ChainTree.Dag.Resolve([]string{"tree", "data", HandlerPath})
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

	supports, _, err := handlerTree.ChainTree.Dag.Resolve([]string{"tree", "data", "jasons-game", "supports"})
	if err != nil {
		return nil, err
	}
	if supports == nil {
		supports = []interface{}{}
	}

	supportedMessages := make(map[string]bool, len(supports.([]interface{})))
	for _, typeString := range supports.([]interface{}) {
		supportedMessages[typeString.(string)] = true
	}

	return &RemoteHandler{
		did:               handlerDid.(string),
		net:               net,
		supportedMessages: supportedMessages,
	}, nil
}
