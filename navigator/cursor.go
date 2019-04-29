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
	locX int
	locY int
}

func (c *Cursor) GetLocation(tree *chaintree.ChainTree) (*Location, error) {
	// log.Printf(spew.Sdump(tree.Get(tree.Tip)))
	pth, remain, err := tree.Dag.Resolve(strings.Split(fmt.Sprintf("tree/data/jasons-game/%d/%d", c.locX, c.locY), "/"))
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
	// log.Printf(spew.Sdump(c.chaintree))

	c.locX = x
	c.locY = y

	return c
}
