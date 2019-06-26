package handlers

import (
	"fmt"

	"github.com/golang/protobuf/proto"
)

type NoopHandler struct {
	Handler
}

func NewNoopHandler() *NoopHandler {
	return &NoopHandler{}
}

func (h *NoopHandler) Send(msg proto.Message) error {
	return fmt.Errorf("Can not send to noop handler")
}

func (h *NoopHandler) Supports(msg proto.Message) bool {
	return false
}

func (h *NoopHandler) SupportsType(msgType string) bool {
	return false
}
