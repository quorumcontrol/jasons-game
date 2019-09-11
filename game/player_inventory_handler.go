package game

import (
	"fmt"
	"sync"

	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/handlers"
	inventoryHandlers "github.com/quorumcontrol/jasons-game/handlers/inventory"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

type PlayerInventoryHandler struct {
	network      network.Network
	did          string
	expectedObjs *sync.Map
	events       *eventstream.EventStream
}

type InventoryChangeEvent struct {
	Did     string
	Action  string
	Message string
	Error   string
}

var PlayerInventoryHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName((*jasonsgame.TransferredObjectMessage)(nil)),
	proto.MessageName((*jasonsgame.RequestObjectTransferMessage)(nil)),
}

func NewPlayerInventoryHandler(network network.Network, playerDid string) *PlayerInventoryHandler {
	return &PlayerInventoryHandler{
		network:      network,
		did:          playerDid,
		expectedObjs: new(sync.Map),
		events:       new(eventstream.EventStream),
	}
}

func (h *PlayerInventoryHandler) Handle(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.RequestObjectTransferMessage:
		err := inventoryHandlers.NewUnrestrictedRemoveHandler(h.network).Handle(msg)
		if err != nil {
			return err
		}
		h.events.Publish(&InventoryChangeEvent{
			Did:    msg.Object,
			Action: "remove",
		})
		return nil
	case *jasonsgame.TransferredObjectMessage:
		var err error

		// This defer ensures either a success or failure is always sent
		defer func() {
			h.expectedObjs.Delete(msg.Object)

			if err != nil {
				h.events.Publish(&InventoryChangeEvent{
					Did:     msg.Object,
					Action:  "add",
					Error:   err.Error(),
				})
			} else {
				h.events.Publish(&InventoryChangeEvent{
					Did:     msg.Object,
					Action:  "add",
					Message: msg.Message,
				})
			}
		}()

		if msg.To != h.did {
			err = fmt.Errorf("Message not intended for this player")
			return err
		}

		isExpected, _ := h.expectedObjs.Load(msg.Object)
		if isExpected == nil || !isExpected.(bool) {
			err = fmt.Errorf("Receive was rejected by player")
			return err
		}

		if msg.Error != "" {
			err = fmt.Errorf(msg.Error)
			return err
		}

		err = inventoryHandlers.NewUnrestrictedAddHandler(h.network).Handle(msg)
		if err != nil {
			return err
		}

		// defer from top will send successful response
		return nil
	default:
		return handlers.ErrUnsupportedMessageType
	}
}

func (h *PlayerInventoryHandler) Supports(msg proto.Message) bool {
	return PlayerInventoryHandlerMessages.Contains(msg)
}

func (h *PlayerInventoryHandler) SupportedMessages() []string {
	return PlayerInventoryHandlerMessages
}

func (h *PlayerInventoryHandler) ExpectObject(did string) {
	h.expectedObjs.Store(did, true)
}

func (h *PlayerInventoryHandler) Subscribe(did string, fn func(changeEvent *InventoryChangeEvent)) *eventstream.Subscription {
	return h.events.Subscribe(func(evt interface{}) {
		switch eMsg := evt.(type) {
		case *InventoryChangeEvent:
			if did == string(eMsg.Did) {
				fn(eMsg)
			}
		}
	})
}

func (h *PlayerInventoryHandler) Unsubscribe(sub *eventstream.Subscription) {
	h.events.Unsubscribe(sub)
}
