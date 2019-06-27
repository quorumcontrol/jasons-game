package inventory

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	broadcastHandlers "github.com/quorumcontrol/jasons-game/handlers/broadcast"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

type UnrestrictedRemoveHandler struct {
	network network.Network
}

var UnrestrictedRemoveHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName((*jasonsgame.RequestTransferObjectMessage)(nil)),
}

func NewUnrestrictedRemoveHandler(network network.Network) handlers.Handler {
	return &UnrestrictedRemoveHandler{
		network: network,
	}
}

func (h *UnrestrictedRemoveHandler) Handle(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.RequestTransferObjectMessage:
		sourceInventory, err := trees.FindInventoryTree(h.network, msg.From)
		if err != nil {
			return fmt.Errorf("error fetching source chaintree: %v", err)
		}

		targetInventory, err := trees.FindInventoryTree(h.network, msg.To)
		if err != nil {
			return fmt.Errorf("error fetching target chaintree: %v", err)
		}

		targetAuths, err := targetInventory.Authentications()
		if err != nil {
			return fmt.Errorf("error fetching target chaintree authentications %s; error: %v", msg.To, err)
		}

		objectTree, err := h.network.GetTree(msg.Object)
		if err != nil {
			return fmt.Errorf("error fetching object chaintree %s: %v", msg.Object, err)
		}

		remoteTargetHandler, err := handlers.FindHandlerForTree(h.network, msg.To)
		if err != nil {
			return fmt.Errorf("error fetching handler for %v", msg.To)
		}
		var targetHandler handlers.Handler
		if remoteTargetHandler != nil {
			targetHandler = remoteTargetHandler
		} else {
			targetHandler = broadcastHandlers.NewTopicBroadcastHandler(h.network, targetInventory.BroadcastTopic())
		}

		transferredObjectMessage := &jasonsgame.TransferredObjectMessage{
			From:   msg.From,
			To:     msg.To,
			Object: msg.Object,
		}

		if !targetHandler.Supports(transferredObjectMessage) {
			return fmt.Errorf("transfer to inventory %v is not supported", msg.To)
		}

		objectTree, err = h.network.ChangeChainTreeOwner(objectTree, targetAuths)
		if err != nil {
			return fmt.Errorf("error changing object owner: %v", err)
		}

		err = sourceInventory.Remove(msg.Object)
		if err != nil {
			return fmt.Errorf("error updating objects in inventory: %v", err)
		}

		if err := targetHandler.Handle(transferredObjectMessage); err != nil {
			return err
		}

		return nil
	default:
		return handlers.ErrUnsupportedMessageType
	}
}

func (h *UnrestrictedRemoveHandler) Supports(msg proto.Message) bool {
	return UnrestrictedRemoveHandlerMessages.Contains(msg)
}

func (h *UnrestrictedRemoveHandler) SupportedMessages() []string {
	return UnrestrictedRemoveHandlerMessages
}
