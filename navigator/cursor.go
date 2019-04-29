package navigator

import (
	"github.com/quorumcontrol/chaintree/chaintree"
)

type Cursor struct {
	tree *chaintree.ChainTree
	locX int
	locY int
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

func (c *Cursor) SetChainTree(tree *chaintree.ChainTree) *Cursor {
	c.tree = tree
	return c
}

func (c *Cursor) GetLocation() (*Location, error) {
	return locationFromTree(c.tree, c.locX, c.locY)
}

func (c *Cursor) SetLocation(x, y int) *Cursor {
	c.locX = x
	c.locY = y

	return c
}
