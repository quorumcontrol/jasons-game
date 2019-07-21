// +build internal

package network

import (
	"crypto/ecdsa"

	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func (n *RemoteNetwork) DisallowReceiveInk(chaintreeId string) {
	// nop in internal builds
}

func (n *RemoteNetwork) ReceiveInk(tree *consensus.SignedChainTree, tokenPayload *transactions.TokenPayload) error {
	return n.receiveInk(tree, n.PrivateKey(), tokenPayload)
}

func (n *RemoteNetwork) ReceiveInkOnEphemeralChainTree(tree *consensus.SignedChainTree, privateKey *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) error {
	return n.receiveInk(tree, privateKey, tokenPayload)
}
