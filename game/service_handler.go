package game

import (
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/jasons-game/network"
)

const HandlerPath = "jasons-game-handler"

type ServiceHandler struct {
	did string
	supportedMessages map[string]bool
}

func (h *ServiceHandler) Did() string {
	return h.did
}

func (h *ServiceHandler) Supports(msgType string) bool {
	return h.supportedMessages[msgType] || false
}

func FindHandlerForTree(net network.Network, tree *consensus.SignedChainTree) (*ServiceHandler, error) {
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

	return &ServiceHandler{
		did: handlerDid.(string),
		supportedMessages: supportedMessages,
	}, nil
}