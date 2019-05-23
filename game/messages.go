//go:generate msgp

package game

import (
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
)

func init() {
	messages.RegisterMessage(&ChatMessage{})
	messages.RegisterMessage(&ShoutMessage{})
	messages.RegisterMessage(&JoinMessage{})
	messages.RegisterMessage(&OpenPortalMessage{})
	messages.RegisterMessage(&OpenPortalResponseMessage{})
	messages.RegisterMessage(&TransferredObjectMessage{})
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
	OnLandId  string
	ToLandId  string
	LocationX int64
	LocationY int64
}

func (cm *OpenPortalMessage) TypeCode() int8 {
	return -103
}

type OpenPortalResponseMessage struct {
	Accepted  bool
	Opener    string
	LandId    string
	LocationX int64
	LocationY int64
}

func (cm *OpenPortalResponseMessage) TypeCode() int8 {
	return -104
}

type TransferredObjectMessage struct {
	From   string
	To     string
	Object string
	Loc    []int64
}

func (m *TransferredObjectMessage) TypeCode() int8 {
	return -105
}
