package game

import (
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
)

func createHome(n network.Network) (*LocationTree, error) {
	log.Debug("creating new home")

	homeTree, err := n.CreateLocalChainTree("home")
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}

	homeLocation := NewLocationTree(n, homeTree)

	err = homeLocation.SetDescription(`You are home in a small cozy, rustic room. There are shelves on the walls where you
can leave things and a fireplace in the corner lit with a warm fire.
This is your space. Not exactly part of fae proper, but also not part of your physical world.

You can use your powers to 'portal to fae' or 'portal to mountain' from here.`)
	if err != nil {
		return nil, errors.Wrap(err, "error setting description on home tree")
	}

	err = homeLocation.AddInteraction(&ChangeNamedLocationInteraction{
		Command: "portal to fae",
		Name:    "last-location",
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to home tree")
	}

	err = homeLocation.AddInteraction(&ChangeNamedLocationInteraction{
		Command: "portal to mountain",
		Name:    "ArcadiaMountain",
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to home tree")
	}

	return homeLocation, nil
}
