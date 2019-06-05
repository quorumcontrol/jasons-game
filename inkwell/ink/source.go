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

	if cfg.ChainTreeDid == "" {
		ct, err = EnsureChainTree(cfg.Net)
	} else {
		ct, err = GetChainTreeByDID(cfg.Net, cfg.ChainTreeDid)
	}

	if err != nil {
		return nil, err
	}

	return &ChainTreeInkSource{
		ct:  ct,
		net: cfg.Net,
	}, nil
}

// EnsureChainTree is for when you want to get or generate a new ink source w/ random DID.
// Not intended for production use (see GetChainTreeByDID for that).
func EnsureChainTree(net network.Network) (*consensus.SignedChainTree, error) {
	existing, err := net.GetChainTreeByName("ink-source")
	if existing == nil {
		if err != nil {
			return nil, errors.Wrap(err, "error checking for existing ink-source chaintree")
		}
		return net.CreateNamedChainTree("ink-source")
	}

	return existing, nil
}

// GetChainTreeByDID is used to obtain a pre-existing ink source given a specific DID.
// Returns an error if it doesn't exist. This is what you want in production.
func GetChainTreeByDID(net network.Network, did string) (*consensus.SignedChainTree, error) {
	return net.GetTree(did)
}

func (ctis *ChainTreeInkSource) RequestInk(amount uint64) (*transactions.ReceiveTokenPayload, error) {
	return nil, nil
}
