package trees

import (
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
)

func createHome(n network.Network) (*LocationTree, error) {
	homeTree, err := n.CreateNamedChainTree("home")
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}

	homeLocation := NewLocationTree(n, homeTree)

	err = homeLocation.SetHandler("did:tupelo:0x4E1109E1d71347165C431297F831e9D1158931b7")
	if err != nil {
		return nil, errors.Wrap(err, "error updating home tree handlers")
	}

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

	err = northLocation.AddInteraction(&ChangeLocationInteraction{
		Command: "south",
		Did:     homeTree.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to north tree")
	}

	err = homeLocation.AddInteraction(&ChangeLocationInteraction{
		Command: "north",
		Did:     northLocation.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to home tree")
	}

	return homeLocation, nil
}
