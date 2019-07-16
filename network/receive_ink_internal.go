// +build !public

package network

import (
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func (n *RemoteNetwork) ReceiveInk(tree *consensus.SignedChainTree, tokenPayload *transactions.TokenPayload) error {
	decodedTip, err := cid.Decode(tokenPayload.Tip)
	if err != nil {
		return errors.Wrapf(err, "error decoding token payload tip: %s", tokenPayload.Tip)
	}

	transaction, err := chaintree.NewReceiveTokenTransaction(tokenPayload.TransactionId, decodedTip.Bytes(), tokenPayload.Signature, tokenPayload.Leaves)
	if err != nil {
		return errors.Wrap(err, "error generating ink receive token transaction")
	}

	_, err = n.Tupelo.PlayTransactions(tree, n.PrivateKey(), []*transactions.Transaction{transaction})
	if err != nil {
		return errors.Wrap(err, "error playing ink receive token transaction")
	}

	err = n.TreeStore().SaveTreeMetadata(tree)
	if err != nil {
		return errors.Wrap(err, "error saving chaintree metadata after ink receive transaction")
	}

	return nil
}
