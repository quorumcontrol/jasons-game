// +build internal

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

	return n.receiveInk(inkWell, n.PrivateKey(), tokenPayload)
}

func (n *RemoteNetwork) ReceiveInkOnEphemeralChainTree(tree *consensus.SignedChainTree, privateKey *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) error {
	return n.receiveInk(tree, privateKey, tokenPayload)
}
