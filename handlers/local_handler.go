package handlers

import (
	"time"
	"github.com/golang/protobuf/proto"
	"github.com/AsynkronIT/protoactor-go/actor"
)

type LocalHandler struct {
	Handler
	pid *actor.PID
}

func NewLocalHandler(pid *actor.PID) *LocalHandler {
	return &LocalHandler{pid: pid}
}

func (h *LocalHandler) Send(msg proto.Message) error {
	actor.EmptyRootContext.Send(h.pid, msg)
	return nil
}

func (h *LocalHandler) Supports(msg proto.Message) bool {
	msgType := proto.MessageName(msg)
	return h.SupportsType(msgType)
}

func (h *LocalHandler) SupportsType(msgType string) bool {
	response, err := actor.EmptyRootContext.RequestFuture(h.pid, &GetSupportedMessages{}, 1 * time.Second).Result()
	if err != nil {
		return false
	}

	for _, msg := range response.([]string) {
		if msg == msgType {
			return true
		}
	}
	return false
}