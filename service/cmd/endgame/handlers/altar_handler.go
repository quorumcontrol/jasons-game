package endgame 

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/jasons-game/game/trees"
  "github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

var log = logging.Logger("endgame")

type EndGameAltarHandler struct {
	network network.Network
	altars  []*EndGameAltar
}

type EndGameAltar struct {
	Did      string
	Requires []string
}

var EndGameAltarHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName((*jasonsgame.TransferredObjectMessage)(nil)),
}

func NewEndGameAltarHandler(network network.Network, altars []*EndGameAltar) handlers.Handler {
	return &EndGameAltarHandler{
		network: network,
		altars:  altars,
	}
}

func (h *EndGameAltarHandler) Handle(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.TransferredObjectMessage:
		err := h.handleTransferredObjectMessage(msg)
		if err != nil {
			log.Error(err)
		}
		return err
	default:
		return handlers.ErrUnsupportedMessageType
	}
}

func (h *EndGameAltarHandler) handleTransferredObjectMessage(msg *jasonsgame.TransferredObjectMessage) error {
	altarDid := msg.To
	objectDid := msg.Object
	playerDid := msg.From

	registeredAltar := false
	for _, altar := range h.altars {
		registeredAltar = registeredAltar || altarDid == altar.Did
	}

	if !registeredAltar {
		return fmt.Errorf("altar not registered")
	}

	// Normal transfer in
	altarInventory, err := trees.FindInventoryTree(h.network, altarDid)
	if err != nil {
		return fmt.Errorf("error fetching inventory chaintree: %v", err)
	}

	altarAuths, err := altarInventory.Authentications()
	if err != nil {
		return fmt.Errorf("error fetching target chaintree authentications %s; error: %v", altarDid, err)
	}

	exists, err := altarInventory.Exists(objectDid)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	err = altarInventory.Add(objectDid)
	if err != nil {
		return err
	}

	objectTree, err := h.network.GetTree(objectDid)
	if err != nil {
		return fmt.Errorf("error fetching object chaintree %s: %v", objectDid, err)
	}

	objectTree, err = h.network.ChangeChainTreeOwner(objectTree, altarAuths)
	if err != nil {
		return fmt.Errorf("error changing object owner: %v", err)
	}

	objectTree, err = h.network.UpdateChainTree(objectTree, "jasons-game/sacrificed-by", playerDid) // nolint
	if err != nil {
		return fmt.Errorf("error changing data: %v", err)
	}

	return nil
}

func (h *EndGameAltarHandler) Supports(msg proto.Message) bool {
	return EndGameAltarHandlerMessages.Contains(msg)
}

func (h *EndGameAltarHandler) SupportedMessages() []string {
	return EndGameAltarHandlerMessages
}
