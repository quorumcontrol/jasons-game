package inventory

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	broadcastHandlers "github.com/quorumcontrol/jasons-game/handlers/broadcast"
)

type UnrestrictedRemoveHandler struct {
	network network.Network
}

var UnrestrictedRemoveHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName(&jasonsgame.TransferObjectMessage{}),
}

func NewUnrestrictedRemoveHandler(network network.Network) handlers.Handler {
	return &UnrestrictedRemoveHandler{
		network: network,
	}
}

func (h *UnrestrictedRemoveHandler) Handle(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.TransferObjectMessage:
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

		objectTree, err := trees.FindObjectTree(h.network, msg.Object)
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

		err = objectTree.ChangeOwner(targetAuths)
		if err != nil {
			return err
		}

		err = sourceInventory.Remove(msg.Object)
		if err != nil {
			return fmt.Errorf("error updating objects in inventory: %v", err)
		}

		// this runs in some external service, and needs to send to the player some
		// transferredObjectMessage
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

func (h *UnrestrictedRemoveHandler) SupportsType(msgType string) bool {
	return UnrestrictedRemoveHandlerMessages.ContainsType(msgType)
}

func (h *UnrestrictedRemoveHandler) SupportedMessages() []string {
	return UnrestrictedRemoveHandlerMessages
}
