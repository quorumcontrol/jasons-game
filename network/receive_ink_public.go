// +build !internal

package network

import (
	"crypto/ecdsa"
	"errors"

	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var cannotReceiveInk map[string]bool

func (n *RemoteNetwork) DisallowReceiveInk(chaintreeId string) {
	if cannotReceiveInk == nil {
		cannotReceiveInk = make(map[string]bool)
	}

	cannotReceiveInk[chaintreeId] = true
}

func (n *RemoteNetwork) ReceiveInk(tree *consensus.SignedChainTree, tokenPayload *transactions.TokenPayload) error {
	if cannotReceiveInk[tree.MustId()] {
		return errors.New("chaintree cannot receive ink")
	}

	return n.receiveInk(tree, n.PrivateKey(), tokenPayload)
}

func (n *RemoteNetwork) ReceiveInkOnEphemeralChainTree(tree *consensus.SignedChainTree, privateKey *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) error {
	return n.receiveInk(tree, privateKey, tokenPayload)
}
