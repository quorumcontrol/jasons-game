package inventory

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

type UnrestrictedAddHandler struct {
	network network.Network
}

var UnrestrictedAddHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName((*jasonsgame.TransferredObjectMessage)(nil)),
}

func NewUnrestrictedAddHandler(network network.Network) handlers.Handler {
	return &UnrestrictedAddHandler{
		network: network,
	}
}

func (h *UnrestrictedAddHandler) Handle(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.TransferredObjectMessage:
		targetInventory, err := trees.FindInventoryTree(h.network, msg.To)
		if err != nil {
			return fmt.Errorf("error fetching inventory chaintree: %v", err)
		}

		exists, err := targetInventory.Exists(msg.Object)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}

		err = targetInventory.Add(msg.Object)
		if err != nil {
			return err
		}
		return nil
	default:
		return handlers.ErrUnsupportedMessageType
	}
}

func (h *UnrestrictedAddHandler) Supports(msg proto.Message) bool {
	return UnrestrictedAddHandlerMessages.Contains(msg)
}

func (h *UnrestrictedAddHandler) SupportedMessages() []string {
	return UnrestrictedAddHandlerMessages
}
