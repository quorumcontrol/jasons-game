package inventory

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	broadcastHandlers "github.com/quorumcontrol/jasons-game/handlers/broadcast"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/messages/build/go/signatures"
)

type UnrestrictedRemoveHandler struct {
	network network.Network
}

var UnrestrictedRemoveHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName((*jasonsgame.RequestObjectTransferMessage)(nil)),
}

func NewUnrestrictedRemoveHandler(network network.Network) handlers.Handler {
	return &UnrestrictedRemoveHandler{
		network: network,
	}
}

func (h *UnrestrictedRemoveHandler) Handle(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.RequestObjectTransferMessage:
		rollbacks := make([]func(error) error, 0)

		handleRollbacks := func(err error, rollbackFuncs []func(error) error) error {
			for _, rbFn := range rollbackFuncs {
				err = rbFn(err)
			}
			return err
		}

		sourceInventory, err := trees.FindInventoryTree(h.network, msg.From)
		if err != nil {
			return fmt.Errorf("error fetching source chaintree: %v", err)
		}

		sourceAuths, err := sourceInventory.Authentications()
		if err != nil {
			return fmt.Errorf("error fetching source chaintree authentications %s; error: %v", msg.From, err)
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

		// Add current owner and target owner while transfer is in progress
		objectTree, err = h.network.ChangeChainTreeOwner(objectTree, append(sourceAuths, targetAuths...))
		if err != nil {
			return fmt.Errorf("error changing object owner: %v", err)
		}
		rollbacks = append(rollbacks, func(err error) error {
			if _, newErr := h.network.ChangeChainTreeOwner(objectTree, sourceAuths); newErr != nil {
				err = errors.Wrap(err, newErr.Error())
			}
			return err
		})

		err = sourceInventory.Remove(msg.Object)
		if err != nil {
			return fmt.Errorf("error updating objects in inventory: %v", handleRollbacks(err, rollbacks))
		}

		future := actor.NewFuture(10 * time.Second)
		pid := actor.EmptyRootContext.Spawn(actor.PropsFromFunc(func(actorCtx actor.Context) {
			switch msg := actorCtx.Message().(type) {
			case *actor.Started:
				actorCtx.Spawn(h.network.NewCurrentStateSubscriptionProps(objectTree.MustId()))
			case *signatures.CurrentState:
				actorCtx.Send(future.PID(), msg)
			}
		}))
		defer actor.EmptyRootContext.Stop(pid)

		if err := targetHandler.Handle(transferredObjectMessage); err != nil {
			return fmt.Errorf("error with target handler: %v", handleRollbacks(err, rollbacks))
		}

		_, err = future.Result()
		if err != nil {
			return fmt.Errorf("error transferring object, receiver never confirmed: %v", handleRollbacks(err, rollbacks))
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
