//go:generate msgp

package game

import "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"

func init() {
	messages.RegisterMessage(&ChatMessage{})
	messages.RegisterMessage(&ShoutMessage{})
	messages.RegisterMessage(&OpenPortalMessage{})
	messages.RegisterMessage(&OpenPortalResponseMessage{})
}

type ChatMessage struct {
	From    string
	Message string
}

func (cm *ChatMessage) TypeCode() int8 {
	return -100
}

type ShoutMessage struct {
	From    string
	Message string
}

func (cm *ShoutMessage) TypeCode() int8 {
	return -101
}

type OpenPortalMessage struct {
	From      string
	LocationX int
	LocationY int
}

func (cm *OpenPortalMessage) TypeCode() int8 {
	return -103
}
