package navigator

import (
	"fmt"
	"strings"

	cbor "github.com/ipfs/go-ipld-cbor"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/typecaster"
)

func init() {
	cbor.RegisterCborType(Location{})
	typecaster.AddType(Location{})
}

type Cursor struct {
	tree *chaintree.ChainTree
	locX int
	locY int
}

func (c *Cursor) SetChainTree(tree *chaintree.ChainTree) *Cursor {
	c.tree = tree
	return c
}

func (c *Cursor) GetLocation() (*Location, error) {
	pth, remain, err := c.tree.Dag.Resolve(strings.Split(fmt.Sprintf("tree/data/jasons-game/%d/%d", c.locX, c.locY), "/"))
	if err != nil {
		return nil, fmt.Errorf("error resolving: %v", err)
	}
	if len(remain) > 0 {
		return nil, fmt.Errorf("error, path remaining: %v", remain)
	}

	l := new(Location)
	err = typecaster.ToType(pth, l)
	if err != nil {
		return nil, fmt.Errorf("error casting: %v", err)
	}
	return l, nil
}

func (c *Cursor) SetLocation(x, y int) *Cursor {
	c.locX = x
	c.locY = y

	return c
}
