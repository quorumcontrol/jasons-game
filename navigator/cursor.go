package navigator

import (
	"fmt"
	"log"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/quorumcontrol/chaintree/chaintree"
)

type cursor struct {
	chaintree *chaintree.ChainTree
	locX      int
	locY      int
}

func (c *cursor) setLocation(x, y int) (string, error) {
	log.Printf(spew.Sdump(c.chaintree))

	tree := c.chaintree.Dag

	log.Printf(spew.Sdump(tree.Get(tree.Tip)))
	pth, remain, err := tree.Resolve(strings.Split(fmt.Sprintf("tree/data/jasons-game/%d/%d", x, y), "/"))
	if err != nil {
		return "", fmt.Errorf("error resolving: %v", err)
	}
	if len(remain) > 0 {
		return "", fmt.Errorf("error, path remaining: %v", remain)
	}
	return pth.(string), nil
}
