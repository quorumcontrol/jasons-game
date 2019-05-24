//go:generate msgp

package network

import "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"

func init() {
	messages.RegisterMessage(&Block{})
}

type Block struct {
	Cid  []byte
	Data []byte
}

func (b *Block) TypeCode() int8 {
	return -110
}

type Join struct {
	Identity string
}

func (j *Join) TypeCode() int8 {
	return -102
}
