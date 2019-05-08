package navigator

import (
	"fmt"
	"strings"

	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/chaintree/typecaster"
)

const defaultDescription = "You are in a baron wasteland type set description <desc> to call this land something"

func locationFromTree(tree *consensus.SignedChainTree, x, y int64) (*jasonsgame.Location, error) {
	pth, remain, err := tree.ChainTree.Dag.Resolve(strings.Split(fmt.Sprintf("tree/data/jasons-game/%d/%d", x, y), "/"))
	if err != nil {
		return nil, fmt.Errorf("error resolving: %v", err)
	}

	if len(remain) > 0 {
		if len(remain) < 2 {
			return &jasonsgame.Location{
				Did:         tree.MustId(),
				Tip:         tree.Tip().String(),
				X:           x,
				Y:           y,
				Description: defaultDescription,
			}, nil
		}

		return nil, fmt.Errorf("error, maybe this isn't a land? path remaining (%d,%d): %v", x, y, remain)
	}

	l := new(jasonsgame.Location)
	err = typecaster.ToType(pth, l)
	if err != nil {
		return nil, fmt.Errorf("error casting: %v", err)
	}
	l.Did = tree.MustId()
	l.Tip = tree.Tip().String()
	l.X = x
	l.Y = y
	return l, nil
}
