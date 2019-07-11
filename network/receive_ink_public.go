// +build public

package network

import (
	"errors"

	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func (n *RemoteNetwork) ReceiveInk(tree *consensus.SignedChainTree, tokenPayload *transactions.TokenPayload) error {
	return errors.New("not allowed")
}
