package inventory

import (
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/utils/stringslice"
)

type UnrestrictedAddHandler struct {
	network network.Network
}

var UnrestrictedAddHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName((*jasonsgame.TransferredObjectMessage)(nil)),
}

func NewUnrestrictedAddHandler(network network.Network) *UnrestrictedAddHandler {
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

		targetAuths, err := targetInventory.Authentications()
		if err != nil {
			return fmt.Errorf("error fetching target chaintree authentications %s; error: %v", msg.To, err)
		}

		exists, err := targetInventory.Exists(msg.Object)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}

		objectTree, err := h.network.GetTree(msg.Object)
		if err != nil {
			return fmt.Errorf("error fetching object chaintree %s: %v", msg.Object, err)
		}

		objectAuths, err := objectTree.Authentications()
		if err != nil {
			return fmt.Errorf("error fetching object chaintree authentications %s; error: %v", msg.Object, err)
		}

		ownerAddr := crypto.PubkeyToAddress(*h.network.PublicKey()).String()
		isOwner := stringslice.Include(objectAuths, ownerAddr)
		if !isOwner {
			return fmt.Errorf("can not transfer %s, current player is not an owner", msg.Object)
		}

		err = targetInventory.Add(msg.Object)
		if err != nil {
			return err
		}

		_, err = h.network.ChangeChainTreeOwner(objectTree, targetAuths)
		if err != nil {
			return fmt.Errorf("error changing object owner: %v", err)
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
