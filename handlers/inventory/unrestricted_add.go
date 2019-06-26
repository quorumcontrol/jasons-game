package inventory

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

type UnrestrictedAddHandler struct {
	network network.Network
}

func NewUnrestrictedAddHandler(network network.Network) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &UnrestrictedAddHandler{
			network: network,
		}
	})
}

func (h *UnrestrictedAddHandler) SupportedMessages() []string {
	return []string{
		proto.MessageName(&jasonsgame.TransferObjectMessage{}),
	}
}

func (h *UnrestrictedAddHandler) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *handlers.GetSupportedMessages:
		actorCtx.Respond(h.SupportedMessages())
	case *jasonsgame.TransferredObjectMessage:
		err := h.handleTransferredObjectMessage(actorCtx, msg)
		if err != nil {
			panic(fmt.Errorf("error on TransferObjectMessage: %v", err))
		}
	}
}

func (h *UnrestrictedAddHandler) handleTransferredObjectMessage(actorCtx actor.Context, msg *jasonsgame.TransferredObjectMessage) error {
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
}
