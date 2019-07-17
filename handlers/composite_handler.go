package handlers

import (
	"github.com/gogo/protobuf/proto"
)

type CompositeHandler struct {
	Handler
	handlerByMessageType map[string][]Handler
	messages             HandlerMessageList
}

func NewCompositeHandler(handlerList []Handler) *CompositeHandler {
	byMessageType := make(map[string][]Handler)

	for _, handler := range handlerList {
		for _, msgType := range handler.SupportedMessages() {
			if byMessageType[msgType] == nil {
				byMessageType[msgType] = []Handler{}
			}
			byMessageType[msgType] = append(byMessageType[msgType], handler)
		}
	}

	messages := make(HandlerMessageList, len(byMessageType))
	i := 0
	for key := range byMessageType {
		messages[i] = key
		i++
	}
	return &CompositeHandler{
		handlerByMessageType: byMessageType,
		messages:             messages,
	}
}

func (h *CompositeHandler) Handle(msg proto.Message) error {
	if !h.Supports(msg) {
		return ErrUnsupportedMessageType
	}
	msgType := proto.MessageName(msg)
	for _, handler := range h.handlerByMessageType[msgType] {
		err := handler.Handle(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *CompositeHandler) Supports(msg proto.Message) bool {
	return h.messages.Contains(msg)
}

func (h *CompositeHandler) SupportedMessages() []string {
	return h.messages
}
