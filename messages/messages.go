//go:generate msgp

package messages

import (
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
)

func init() {
	messages.RegisterMessage(&ChatMessage{})
	messages.RegisterMessage(&ShoutMessage{})
	messages.RegisterMessage(&OpenPortalMessage{})
	messages.RegisterMessage(&OpenPortalResponseMessage{})
	messages.RegisterMessage(&TransferredObjectMessage{})
	messages.RegisterMessage(&QuestMessage{})
}

type PlayerMessage interface {
	messages.WireMessage

	FromPlayer() string
	ToDid() string
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
	To        string
	ToLandId  string
	LocationX int64
	LocationY int64
}

func (m *OpenPortalMessage) TypeCode() int8 {
	return -103
}

func (m *OpenPortalMessage) FromPlayer() string {
	return m.From
}

func (m *OpenPortalMessage) ToDid() string {
	return m.To
}

type OpenPortalResponseMessage struct {
	From string
	To   string

	Accepted  bool
	Opener    string
	LandId    string
	LocationX int64
	LocationY int64
}

func (cm *OpenPortalResponseMessage) TypeCode() int8 {
	return -104
}

func (m *OpenPortalResponseMessage) FromPlayer() string {
	return m.From
}

func (m *OpenPortalResponseMessage) ToDid() string {
	return m.To
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

type QuestMessage struct {
	Message string
}

func (qm *QuestMessage) TypeCode() int8 {
	return -106
}
