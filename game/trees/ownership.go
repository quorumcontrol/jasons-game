package trees

import (
	"context"
	"fmt"
	"sort"

	"github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/jasons-game/utils/stringslice"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

// VerifyOwnership checks that all expectedAuths currently own the tree
func VerifyOwnership(ctx context.Context, tree *chaintree.ChainTree, expectedAuths []string) (bool, error) {
	auths, err := consensus.NewSignedChainTreeFromChainTree(tree).Authentications()
	if err != nil {
		return false, err
	}

	isValid := stringslice.All(expectedAuths, func(s string) bool {
		return stringslice.Include(auths, s)
	})
	return isValid, nil
}

// VerifyOwnershipAt checks a all expectedAuths owned the tree at a given height
func VerifyOwnershipAt(ctx context.Context, tree *chaintree.ChainTree, height int, expectedAuths []string) (bool, error) {
	treeAt, err := AtHeight(ctx, tree, height)
	if err != nil {
		return false, err
	}
	return VerifyOwnership(ctx, treeAt, expectedAuths)
}

type OwnershipChange struct {
	Tip             cid.Cid
	Authentications []string
	Height          uint64
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

		blockTransactionsUncast, _, err := treeAt.Dag.Resolve(ctx, []string{"chain", "end", "transactions"})
		if err != nil {
			return ownershipChanges, err
		}

		blockTransactionsSliceUncast, ok := blockTransactionsUncast.([]interface{})
		if !ok {
			return ownershipChanges, fmt.Errorf("block transactions is not an array")
		}

		var hasSetOwnership bool
		for _, transactionUncast := range blockTransactionsSliceUncast {
			transaction := &transactions.Transaction{}
			err = typecaster.ToType(transactionUncast, transaction)
			if err != nil {
				return ownershipChanges, err
			}
			if transaction.SetOwnershipPayload != nil {
				hasSetOwnership = true
			}
		}

		if hasSetOwnership {
			auths, err := consensus.NewSignedChainTreeFromChainTree(treeAt).Authentications()
			if err != nil {
				return ownershipChanges, err
			}
			sort.Strings(auths)

			ownershipChanges = append(ownershipChanges, &OwnershipChange{
				Tip:             tip,
				Authentications: auths,
				Height:          MustHeight(ctx, treeAt),
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
