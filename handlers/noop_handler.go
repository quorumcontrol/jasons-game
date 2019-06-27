package handlers

import (
	"github.com/golang/protobuf/proto"
)

type NoopHandler struct {
	Handler
}

func NewNoopHandler() *NoopHandler {
	return &NoopHandler{}
}

func (h *NoopHandler) Handle(msg proto.Message) error {
	return ErrUnsupportedMessageType
}

func (h *NoopHandler) Supports(msg proto.Message) bool {
	return false
}

func (h *NoopHandler) SupportedMessages() []string {
	return []string{}
}
