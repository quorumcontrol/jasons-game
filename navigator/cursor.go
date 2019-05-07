package navigator

import (
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

type Cursor struct {
	tree *consensus.SignedChainTree
	did  string
	locX int64
	locY int64
}

func (c *Cursor) X() int64 {
	return c.locX
}

func (c *Cursor) Y() int64 {
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

func (c *Cursor) GetLocation() (*jasonsgame.Location, error) {
	return locationFromTree(c.tree, c.locX, c.locY)
}

func (c *Cursor) SetLocation(x, y int64) *Cursor {
	c.locX = x
	c.locY = y

	return c
}
