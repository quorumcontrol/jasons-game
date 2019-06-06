package ink

import (
	"github.com/pkg/errors"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/network"
)

type Source interface {
	RequestInk(amount uint64) (*transactions.ReceiveTokenPayload, error)
}

type ChainTreeInkSource struct {
	ct  *consensus.SignedChainTree
	net network.Network
}

type ChainTreeInkSourceConfig struct {
	Net          network.Network
	ChainTreeDid string
}

var _ Source = &ChainTreeInkSource{}

func NewChainTreeInkSource(cfg ChainTreeInkSourceConfig) (*ChainTreeInkSource, error) {
	var (
		ct *consensus.SignedChainTree
		err error
	)

	ct, err = ensureChainTree(cfg.Net)

	if err != nil {
		return nil, err
	}

	ctis := &ChainTreeInkSource{
		ct:  ct,
		net: cfg.Net,
	}

	return ctis, nil
}

// ensureChainTree gets or creates a new ink-source chaintree.
// Note that this chaintree doesn't typically own the ink token; it just contains some
// that was sent to it.
func ensureChainTree(net network.Network) (*consensus.SignedChainTree, error) {
	existing, err := net.GetChainTreeByName("ink-source")
	if existing == nil {
		if err != nil {
			return nil, errors.Wrap(err, "error checking for existing ink-source chaintree")
		}
		return net.CreateNamedChainTree("ink-source")
	}

	return existing, nil
}

func (ctis *ChainTreeInkSource) RequestInk(amount uint64) (*transactions.ReceiveTokenPayload, error) {
	return nil, nil
}
