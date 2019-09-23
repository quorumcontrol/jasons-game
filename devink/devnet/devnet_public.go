// +build !internal

package devnet

import (
	"crypto/ecdsa"
	"errors"

	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/network"
)

// This is unused in public builds, but the compiler wants the types & funcs to exist

type DevRemoteNetwork struct {
	*network.RemoteNetwork
}

func (n *DevRemoteNetwork) PlayTransactionsWithResp(tree *consensus.SignedChainTree, key *ecdsa.PrivateKey, transactions []*transactions.Transaction) (*consensus.SignedChainTree, *consensus.AddBlockResponse, error) {
	return nil, nil, errors.New("unavailable in public build")
}
