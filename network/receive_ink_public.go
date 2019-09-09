// +build !internal

package network

import (
	"crypto/ecdsa"

	"github.com/pkg/errors"

	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func (n *RemoteNetwork) ReceiveInk(tokenPayload *transactions.TokenPayload) error {
	inkWell, err := n.inkWell()
	if err != nil {
		return errors.Wrap(err, "error fetching inkwell")
	}

	return n.receiveInk(inkWell, n.signingKey, tokenPayload)
}

func (n *RemoteNetwork) DepositInk(source *consensus.SignedChainTree, key *ecdsa.PrivateKey, amount uint64) error {
	well, err := n.inkWell()
	if err != nil {
		return errors.Wrap(err, "error fetching inkwell")
	}

	tokenPayload, err := n.Tupelo.SendInk(source, key, amount, well.MustId())
	if err != nil {
		return errors.Wrap(err, "error getting token payload for ink send")
	}

	err = n.TreeStore().SaveTreeMetadata(source)
	if err != nil {
		return errors.Wrap(err, "error saving chaintree metadata after ink send transaction")
	}

	return n.receiveInk(well, n.signingKey, tokenPayload)
}

func (n *RemoteNetwork) ReceiveInkOnEphemeralChainTree(tree *consensus.SignedChainTree, privateKey *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) error {
	return n.receiveInk(tree, privateKey, tokenPayload)
}
