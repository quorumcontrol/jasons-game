package network

import (
	"crypto/ecdsa"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func (n *RemoteNetwork) receiveInk(tree *consensus.SignedChainTree, privateKey *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) error {
	decodedTip, err := cid.Decode(tokenPayload.Tip)
	if err != nil {
		return errors.Wrapf(err, "error decoding token payload tip: %s", tokenPayload.Tip)
	}

	log.Debugf("receive ink decoded token payload tip: %s", decodedTip)

	transaction, err := chaintree.NewReceiveTokenTransaction(tokenPayload.TransactionId, decodedTip.Bytes(), tokenPayload.Signature, tokenPayload.Leaves)
	if err != nil {
		return errors.Wrap(err, "error generating ink receive token transaction")
	}

	log.Debugf("receive ink transaction: %+v", *transaction)

	txResp, err := n.Tupelo.PlayTransactions(tree, privateKey, []*transactions.Transaction{transaction})
	if err != nil {
		return errors.Wrap(err, "error playing ink receive token transaction")
	}

	log.Debugf("receive ink PlayTransactions response: %+v", *txResp)

	err = n.TreeStore().SaveTreeMetadata(tree)
	if err != nil {
		return errors.Wrap(err, "error saving chaintree metadata after ink receive transaction")
	}

	log.Debug("receive ink saved tree metadata")

	return nil
}
