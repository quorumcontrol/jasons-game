//go:generate msgp

package game

import "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"

func init() {
	messages.RegisterMessage(&ChatMessage{})
	messages.RegisterMessage(&ShoutMessage{})
	messages.RegisterMessage(&JoinMessage{})
	messages.RegisterMessage(&OpenPortalMessage{})
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

type JoinMessage struct {
	From string
}

func (cm *JoinMessage) TypeCode() int8 {
	return -102
}

type OpenPortalMessage struct {
	From      string
	LocationX int
	LocationY int
}

func (cm *OpenPortalMessage) TypeCode() int8 {
	return -103
}
