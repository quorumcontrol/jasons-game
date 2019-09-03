package network

import (
	"context"
	"crypto/ecdsa"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func (n *RemoteNetwork) receiveInk(tree *consensus.SignedChainTree, privateKey *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) error {
	transaction, err := n.Tupelo.ReceiveInkTransaction(tree, privateKey, tokenPayload)
	if err != nil {
		return errors.Wrap(err, "error generating ink receive token transaction")
	}

	log.Debugf("receive ink transaction: %+v", transaction)

	txResp, err := n.Tupelo.PlayTransactions(tree, privateKey, []*transactions.Transaction{transaction})
	if err != nil {
		return errors.Wrap(err, "error playing ink receive token transaction")
	}

	log.Debugf("receive ink PlayTransactions response: %+v", txResp)

	err = n.TreeStore().SaveTreeMetadata(tree)
	if err != nil {
		return errors.Wrap(err, "error saving chaintree metadata after ink receive transaction")
	}

	log.Debug("receive ink saved tree metadata")

	log.Debugf("ink faucet chaintree after receive:\n%s", tree.ChainTree.Dag.Dump(context.TODO()))

	return nil
}
