package navigator

import (
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

type Cursor struct {
	tree *consensus.SignedChainTree
	did  string
	locX int
	locY int
}

func (c *Cursor) X() int {
	return c.locX
}

func (c *Cursor) Y() int {
	return c.locY
}

func (c *Cursor) North() *Cursor {
	c.locY++
	return c
}

func (c *Cursor) South() *Cursor {
	c.locY--
	return c
}

func (c *Cursor) East() *Cursor {
	c.locX++
	return c
}

func (c *Cursor) West() *Cursor {
	c.locX--
	return c
}

func (c *Cursor) SetChainTree(tree *consensus.SignedChainTree) *Cursor {
	c.tree = tree
	c.did = tree.MustId()
	return c
}

func (c *Cursor) Did() string {
	return c.did
}

func (c *Cursor) GetLocation() (*Location, error) {
	return locationFromTree(c.tree, c.locX, c.locY)
}

func (c *Cursor) SetLocation(x, y int) *Cursor {
	c.locX = x
	c.locY = y

	return c
}
