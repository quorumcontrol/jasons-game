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

	err = homeLocation.SetDescription("You are in the main hall of Hogwarts. As you observe your surroundings, you see a particularly interesting gargoyle statue on the wall")
	if err != nil {
		return nil, errors.Wrap(err, "error updating home tree")
	}

	healingTree, err := n.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}
	healingLocation := NewLocationTree(n, healingTree)
	err = healingLocation.SetDescription("You took a critical hit and were rushed to the temple of healing. Maybe next time you should fight the dragon with courage.")
	if err != nil {
		return nil, errors.Wrap(err, "error updating healing tree")
	}
	err = healingLocation.AddInteraction(&ChangeLocationInteraction{
		Command: "leave the temple",
		Did:     homeLocation.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to cipher tree")
	}

	hiddenTree, err := n.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}
	hiddenLocation := NewLocationTree(n, hiddenTree)
	err = hiddenLocation.SetDescription("The gargoyle slides aside and you enter the headmaster's chambers. Upon entering, you see a giant dragon staring you down.")
	if err != nil {
		return nil, errors.Wrap(err, "error updating hidden tree")
	}
	err = hiddenLocation.AddInteraction(&ChangeLocationInteraction{
		Command: "run away",
		Did:     homeLocation.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to cipher tree")
	}

	cmd2 := "fight the dragon"
	secret2 := "with courage"
	successInteraction2 := &RespondInteraction{
		Response: "you killed the dragon, now what?",
	}
	failureInteraction2 := &ChangeLocationInteraction{
		Did: healingLocation.MustId(),
	}
	ci2, err := NewCipherInteraction(cmd2, secret2, successInteraction2, failureInteraction2)
	if err != nil {
		return nil, errors.Wrap(err, "cipher tree error")
	}
	err = hiddenLocation.AddInteraction(ci2)
	if err != nil {
		return nil, errors.Wrap(err, "cipher tree error")
	}

	cmd := "whisper to the gargoyle"
	secret := "sherbert lemon"
	failureInteraction := &RespondInteraction{
		Response: "Nothing happens. You remember professor McGonagall uttering a specific phrase to enter Dumbledore's chambers, maybe you should try that.",
	}
	successInteraction := &ChangeLocationInteraction{
		Did: hiddenLocation.MustId(),
	}
	ci, err := NewCipherInteraction(cmd, secret, successInteraction, failureInteraction)
	if err != nil {
		return nil, errors.Wrap(err, "cipher tree error")
	}
	err = homeLocation.AddInteraction(ci)
	if err != nil {
		return nil, errors.Wrap(err, "cipher tree error")
	}

	return homeLocation, nil
}
