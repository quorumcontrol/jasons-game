package trees

import (
	"context"
	"fmt"
	"sort"

	"github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/jasons-game/utils/stringslice"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

type OwnershipChange struct {
	Tip             cid.Cid
	Authentications []string
}

// OwnershipChanges returns a slice of OwnershipChange objects from newest to oldest
func OwnershipChanges(ctx context.Context, tree *chaintree.ChainTree) ([]*OwnershipChange, error) {
	var err error

	ownershipChanges := []*OwnershipChange{}
	tip := tree.Dag.Tip

	if err != nil {
		return ownershipChanges, err
	}

	for !tip.Equals(cid.Undef) {
		treeAt, err := tree.At(ctx, &tip)
		if err != nil {
			return ownershipChanges, err
		}

		auths, err := consensus.NewSignedChainTreeFromChainTree(treeAt).Authentications()
		if err != nil {
			return ownershipChanges, err
		}
		sort.Strings(auths)

		var didChange bool
		if len(ownershipChanges) == 0 {
			didChange = true
		} else {
			lastChange := ownershipChanges[len(ownershipChanges)-1]
			didChange = !stringslice.Equal(auths, lastChange.Authentications)
		}

		if didChange {
			ownershipChanges = append(ownershipChanges, &OwnershipChange{
				Tip:             tip,
				Authentications: auths,
			})
		}

		previousBlockUncast, _, err := treeAt.Dag.Resolve(ctx, []string{"chain", "end"})
		if err != nil {
			return ownershipChanges, err
		}
		if previousBlockUncast == nil {
			break
		}
		previousBlock, ok := previousBlockUncast.(map[string]interface{})
		if !ok {
			return ownershipChanges, fmt.Errorf("chain previous block could not be cast")
		}

		previousTip, ok := previousBlock["previousTip"]
		if !ok || previousTip == nil {
			break
		}

		tip, ok = previousTip.(cid.Cid)
		if !ok {
			return ownershipChanges, fmt.Errorf("chain.end could not be cast")
		}
	}

	return ownershipChanges, err
}
