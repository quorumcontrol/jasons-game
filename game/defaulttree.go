package game

import (
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
)

func createHome(n network.Network) (*consensus.SignedChainTree, error) {
	log.Debug("creating new home")
	tree, err := n.CreateNamedChainTree("home")
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}
	log.Debug("updating home")
	tree, err = n.UpdateChainTree(tree, "jasons-game/0/0", &navigator.Location{Description: "hi, welcome"})
	if err != nil {
		return nil, errors.Wrap(err, "error updating tree")
	}
	return n.UpdateChainTree(tree, "jasons-game/0/1", &navigator.Location{Description: "north of welcome"})
}
