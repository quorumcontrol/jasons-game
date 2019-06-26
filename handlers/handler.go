package handlers

import (
	"github.com/golang/protobuf/proto"
)

const HandlerPath = "jasons-game-handler"

type Handler interface {
	Send(proto.Message) error
	Supports(proto.Message) bool
	SupportsType(string) bool
}

type GetSupportedMessages struct{}