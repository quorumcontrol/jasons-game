package handlers

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/require"
)

type fakeHandler struct {
	messages HandlerMessageList
	expected func(proto.Message)
}

func (h *fakeHandler) Handle(msg proto.Message) error {
	h.expected(msg)
	return nil
}

func (h *fakeHandler) Supports(msg proto.Message) bool {
	return h.messages.Contains(msg)
}

func (h *fakeHandler) SupportedMessages() []string {
	return h.messages
}

func TestCompositeHandler(t *testing.T) {
	var expected1 proto.Message
	handler1Msg := ptypes.TimestampNow()
	handler1 := &fakeHandler{
		messages: HandlerMessageList{proto.MessageName(handler1Msg)},
		expected: func(msg proto.Message) {
			expected1 = msg
		},
	}

	var expected2 proto.Message
	handler2Msg := ptypes.DurationProto(time.Duration(1))
	handler2 := &fakeHandler{
		messages: HandlerMessageList{proto.MessageName(handler2Msg)},
		expected: func(msg proto.Message) {
			expected2 = msg
		},
	}

	compositeHandler := NewCompositeHandler([]Handler{handler1, handler2})
	compositeHandler.Handle(handler1Msg)
	require.Equal(t, expected1, handler1Msg)
	require.Nil(t, expected2)

	compositeHandler.Handle(handler2Msg)
	require.Equal(t, expected2, handler2Msg)
}
