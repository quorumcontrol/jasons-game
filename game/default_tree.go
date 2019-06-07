package game

import (
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
)

func createHome(n network.Network) (*LocationTree, error) {
	log.Debug("creating new home")

	homeTree, err := n.CreateNamedChainTree("home")
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}

	homeLocation := NewLocationTree(n, homeTree)

	err = homeLocation.SetDescription("hi, welcome")
	if err != nil {
		return nil, errors.Wrap(err, "error updating home tree")
	}

	northTree, err := n.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}

	northLocation := NewLocationTree(n, northTree)
	err = northLocation.SetDescription("north of welcome")
	if err != nil {
		return nil, errors.Wrap(err, "error updating north tree")
	}

	err = northLocation.AddInteraction(&Interaction{
		Command: "south",
		Action:  "changeLocation",
		Args: map[string]string{
			"did": homeTree.MustId(),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to north tree")
	}

	err = homeLocation.AddInteraction(&Interaction{
		Command: "north",
		Action:  "changeLocation",
		Args: map[string]string{
			"did": northLocation.MustId(),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to home tree")
	}

	return homeLocation, nil
}
