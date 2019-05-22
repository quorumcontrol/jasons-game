//go:generate msgp

package game

import "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"

func init() {
	messages.RegisterMessage(&ChatMessage{})
	messages.RegisterMessage(&ShoutMessage{})
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

}
