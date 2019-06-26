package handlers

import (
	"github.com/golang/protobuf/proto"
)

type CompositHandler struct {
	Handler
	handlerByMessageType map[string][]Handler
	messages             HandlerMessageList
}

func NewCompositHandler(handlerList []Handler) *CompositHandler {
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
	return &CompositHandler{
		handlerByMessageType: byMessageType,
		messages:             messages,
	}
}

func (h *CompositHandler) Handle(msg proto.Message) error {
	msgType := proto.MessageName(msg)
	if !h.Supports(msg) {
		return ErrUnsupportedMessageType
	}
	for _, handler := range h.handlerByMessageType[msgType] {
		err := handler.Handle(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *CompositHandler) Supports(msg proto.Message) bool {
	return h.messages.Contains(msg)
}

func (h *CompositHandler) SupportedMessages() []string {
	return h.messages
}
