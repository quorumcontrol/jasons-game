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

type UnrestrictedRemoveHandler struct {
	network network.Network
}

func NewUnrestrictedRemoveHandler(network network.Network) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &UnrestrictedRemoveHandler{
			network: network,
		}
	})
}

func (h *UnrestrictedRemoveHandler) SupportedMessages() []string {
	return []string{
		proto.MessageName(&jasonsgame.TransferObjectMessage{}),
	}
}

func (h *UnrestrictedRemoveHandler) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *handlers.GetSupportedMessages:
		actorCtx.Respond(h.SupportedMessages())
	case *jasonsgame.TransferObjectMessage:
		err := h.handleTransferObjectMessage(actorCtx, msg)
		if err != nil {
			panic(fmt.Errorf("error on TransferObjectMessage: %v", err))
		}
	}
}

func (h *UnrestrictedRemoveHandler) handleTransferObjectMessage(actorCtx actor.Context, msg *jasonsgame.TransferObjectMessage) error {
	sourceInventory, err := trees.FindInventoryTree(h.network, msg.From)
	if err != nil {
		return fmt.Errorf("error fetching source chaintree: %v", err)
	}

	targetTree, err := h.network.GetTree(msg.To)
	if err != nil {
		return err
	}

	targetAuths, err := targetTree.Authentications()
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
		// TODO: FIXME to use remove handler
		targetHandler = handlers.NewLocalHandler(actorCtx.Self())
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

	if err := targetHandler.Send(transferredObjectMessage); err != nil {
		return err
	}

	return nil
}
