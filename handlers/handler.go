package handlers

import (
	"errors"

	"github.com/golang/protobuf/proto"
)

const HandlerPath = "jasons-game-handler"

var ErrUnsupportedMessageType = errors.New("message type is not supported")

type Handler interface {
	Handle(proto.Message) error
	Supports(proto.Message) bool
	SupportsType(string) bool
	SupportedMessages() []string
}
