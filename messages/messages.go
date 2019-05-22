//go:generate msgp

package messages

import "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"

func init() {
	messages.RegisterMessage(&OpenPortalMessage{})
	messages.RegisterMessage(&OpenPortalResponseMessage{})
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
