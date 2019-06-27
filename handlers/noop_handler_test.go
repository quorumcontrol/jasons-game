package handlers

import (
	"testing"

	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

func TestNoopHandler(t *testing.T) {
	h := NewNoopHandler()
	require.False(t, h.Supports(&jasonsgame.ChatMessage{}))
	require.Equal(t, h.SupportedMessages(), []string{})
	require.Equal(t, h.Handle(&jasonsgame.ChatMessage{}), ErrUnsupportedMessageType)
}
