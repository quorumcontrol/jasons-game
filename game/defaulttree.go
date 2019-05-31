package game

import (
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func createHome(n network.Network) (*consensus.SignedChainTree, error) {
	log.Debug("creating new home")
	tree, err := n.CreateNamedChainTree("home")
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}
	log.Debug("updating home")
	return n.UpdateChainTree(tree, "jasons-game/description", "hi, welcome")
}
