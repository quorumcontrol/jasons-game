//go:generate msgp

package network

import "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"

func init() {
	messages.RegisterMessage(&Block{})
}

type Block struct {
	Cid []byte
}

func (b *Block) TypeCode() int8 {
	return -110
}
