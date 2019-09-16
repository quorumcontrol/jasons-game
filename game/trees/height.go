package trees

import (
	"context"
	"fmt"

	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
)

// MustHeight gets the height of a chaintree, panics on error
func MustHeight(ctx context.Context, tree *chaintree.ChainTree) uint64 {
	treeHeight, err := Height(ctx, tree)
	if err != nil {
		panic(err)
	}
	return treeHeight
}

// Height gets the height of a chaintree
func Height(ctx context.Context, tree *chaintree.ChainTree) (uint64, error) {
	rootNode := chaintree.RootNode{}
	err := tree.Dag.ResolveInto(ctx, []string{""}, &rootNode)

	if err != nil {
		return 0, errors.Wrap(err, "error resolving tree height")
	}

	return rootNode.Height, nil
}

// AtHeight returns the ChainTree at a given height
func AtHeight(ctx context.Context, tree *chaintree.ChainTree, height uint64) (*chaintree.ChainTree, error) {
	tip := tree.Dag.Tip
	for !tip.Equals(cid.Undef) {
		treeAt, err := tree.At(ctx, &tip)
		if err != nil {
			return nil, err
		}

		treeHeight, err := Height(ctx, treeAt)
		if err != nil {
			return nil, err
		}

		if uint64(height) == treeHeight {
			return treeAt, nil
		}

		// if the expected height is greater than our first found height,
		// we will never find the specified height, so exit out
		if uint64(height) > treeHeight {
			return nil, fmt.Errorf("height %d is out of range", height)
		}

		previousBlockUncast, _, err := treeAt.Dag.Resolve(ctx, []string{"chain", "end"})
		if err != nil {
			return nil, err
		}
		if previousBlockUncast == nil {
			break
		}
		previousBlock, ok := previousBlockUncast.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("chain previous block could not be cast")
		}

		previousTip, ok := previousBlock["previousTip"]
		if !ok || previousTip == nil {
			break
		}

		tip, ok = previousTip.(cid.Cid)
		if !ok {
			return nil, fmt.Errorf("chain.end could not be cast")
		}
	}

	return nil, fmt.Errorf("height of %d not found", height)
}
