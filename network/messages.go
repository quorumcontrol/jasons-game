//go:generate msgp

package network

type Block struct {
	Cid []byte
}

func (b *Block) TypeCode() int8 {
	return -110
}
