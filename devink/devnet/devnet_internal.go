// +build internal

package devnet

import (
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/pkg/errors"
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
	well, err := n.RemoteNetwork.InkWell()
	if err != nil {
		return nil, nil, errors.Wrap(err, "error fetching remote network inkwell")
	}

	txResp, err := n.Tupelo.PlayTransactions(tree, n.PrivateKey(), transactions, well)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error playing transaction")
	}

	err = n.TreeStore().SaveTreeMetadata(tree)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error saving tree metadata")
	}

	return tree, txResp, nil
}
