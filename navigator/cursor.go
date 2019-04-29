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

type cursor struct {
	chaintree *chaintree.ChainTree
	locX      int
	locY      int
}

// Location is the representation of a grid element
type Location struct {
	Description string
}

func (c *cursor) setLocation(x, y int) (*Location, error) {
	// log.Printf(spew.Sdump(c.chaintree))

	tree := c.chaintree.Dag

	// log.Printf(spew.Sdump(tree.Get(tree.Tip)))
	pth, remain, err := tree.Resolve(strings.Split(fmt.Sprintf("tree/data/jasons-game/%d/%d", x, y), "/"))
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
