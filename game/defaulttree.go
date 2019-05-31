package game

import (
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func createHome(n network.Network) (*consensus.SignedChainTree, error) {
	log.Debug("creating new home")
	homeTree, err := n.CreateNamedChainTree("home")
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}

	northTree, err := n.CreateNamedChainTree(homeTree.MustId() + "/north")
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}
	northTree, err = n.UpdateChainTree(northTree, "jasons-game/description", "north of welcome")
	if err != nil {
		return nil, errors.Wrap(err, "error updating tree")
	}
	northTree, err = n.UpdateChainTree(northTree, "jasons-game/interactions/south", map[string]interface{}{
		"action": "changeLocation",
		"args": map[string]string{
			"did": homeTree.MustId(),
		},
	})

	log.Debug("updating home")
	homeTree, err = n.UpdateChainTree(homeTree, "jasons-game/interactions/north", map[string]interface{}{
		"action": "changeLocation",
		"args": map[string]string{
			"did": northTree.MustId(),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "error updating tree")
	}

	return n.UpdateChainTree(homeTree, "jasons-game/description", "hi, welcome")
}
