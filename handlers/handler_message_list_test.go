package handlers

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

func TestHandlerMessageList(t *testing.T) {
	testList := HandlerMessageList{
		proto.MessageName((*jasonsgame.ChatMessage)(nil)),
	}

	require.True(t, testList.Contains(&jasonsgame.ChatMessage{}))
	require.False(t, testList.Contains(ptypes.TimestampNow()))

	require.True(t, testList.ContainsType(proto.MessageName(&jasonsgame.ChatMessage{})))
	require.False(t, testList.ContainsType(proto.MessageName(ptypes.TimestampNow())))
}
