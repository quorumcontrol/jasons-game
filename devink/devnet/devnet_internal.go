// +build internal

package devnet

import (
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/network"
)

type DevRemoteNetwork struct {
	*network.RemoteNetwork
}

type DevNetwork interface {
	network.Network
	PlayTransactionsWithResp(tree *consensus.SignedChainTree, transactions []*transactions.Transaction) (*consensus.SignedChainTree, *consensus.AddBlockResponse, error)
}

var _ DevNetwork = &DevRemoteNetwork{}

func (n *DevRemoteNetwork) PlayTransactionsWithResp(tree *consensus.SignedChainTree, transactions []*transactions.Transaction) (*consensus.SignedChainTree, *consensus.AddBlockResponse, error) {
	txResp, err := n.Tupelo.PlayTransactions(tree, n.PrivateKey(), transactions)
	if err != nil {
		return nil, nil, err
	}

	err = n.TreeStore().SaveTreeMetadata(tree)
	if err != nil {
		return nil, nil, err
	}

	return tree, txResp, nil
}
