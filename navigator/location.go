package navigator

import (
	"fmt"
	"strings"

	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/chaintree/typecaster"
)

func init() {
	cbor.RegisterCborType(Location{})
	typecaster.AddType(Location{})
}

const defaultDescription = "You are in a baron wasteland type set description <desc> to call this land something"

// Location is the representation of a grid element
type Location struct {
	Did         string
	X           int
	Y           int
	Description string
}

func locationFromTree(tree *consensus.SignedChainTree, x, y int) (*Location, error) {
	pth, remain, err := tree.ChainTree.Dag.Resolve(strings.Split(fmt.Sprintf("tree/data/jasons-game/%d/%d", x, y), "/"))
	if err != nil {
		return nil, fmt.Errorf("error resolving: %v", err)
	}

	if len(remain) > 0 {
		if len(remain) < 2 {
			return &Location{
				Did:         tree.MustId(),
				X:           x,
				Y:           y,
				Description: defaultDescription,
			}, nil
		}

		return nil, fmt.Errorf("error, maybe this isn't a land? path remaining (%d,%d): %v", x, y, remain)
	}

	l := new(Location)
	err = typecaster.ToType(pth, l)
	if err != nil {
		return nil, fmt.Errorf("error casting: %v", err)
	}
	l.Did = tree.MustId()
	l.X = x
	l.Y = y
	return l, nil
}
