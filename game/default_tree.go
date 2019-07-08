package game

import (
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/game/trees"
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
		return nil, errors.Wrap(err, "error updating hogwarts tree")
	}

	headmastersChambersTree, err := n.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating headmasters tree")
	}
	headmastersChambersLocation := NewLocationTree(n, headmastersChambersTree)
	err = headmastersChambersLocation.SetDescription("The gargoyle slides aside and you enter the headmaster's chambers. Upon entering, you see a giant dragon staring you down.")
	if err != nil {
		return nil, errors.Wrap(err, "error updating headmasters tree")
	}
	err = headmastersChambersLocation.AddInteraction(&ChangeLocationInteraction{
		Command: "run away",
		Did:     homeLocation.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to headmasters tree")
	}
	err = headmastersChambersLocation.AddInteraction(&ChangeLocationInteraction{
		Command: "return to Hogwarts' main hall",
		Did:     homeLocation.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to headmasters tree")
	}
	headmastersChambersInventory := trees.NewInventoryTree(n, headmastersChambersTree)

	gargoyleTree, err := n.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating gargoyle tree")
	}
  gargoyleObj := NewObjectTree(n, gargoyleTree)
  err = gargoyleObj.SetName("gargoyle-statue")
  if err != nil {
		return nil, errors.Wrap(err, "error setting name of new object")
  }
  gargoyleWhisperInteraction, err := NewCipherInteraction(
		"whisper to the gargoyle", "sherbert lemon",
		&ChangeLocationInteraction{
			Did: headmastersChambersLocation.MustId(),
		},
		&RespondInteraction{
			Response: "Nothing happens. You remember professor McGonagall uttering a specific phrase to enter Dumbledore's chambers, maybe you should try that.",
		},
	)
	err = gargoyleObj.AddInteraction(gargoyleWhisperInteraction)
	if err != nil {
		return nil, errors.Wrap(err, "cipher tree error")
	}

  homeInventory := trees.NewInventoryTree(n, homeTree)
  err = homeInventory.Add(gargoyleObj.MustId())
  if err != nil {
		return nil, errors.Wrap(err, "error adding gargoyle to home")
	}

	healingTempleTree, err := n.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}
	healingTempleLocation := NewLocationTree(n, healingTempleTree)
	err = healingTempleLocation.SetDescription("You took a critical hit and were rushed to the temple of healing. Maybe next time you should fight the dragon with courage.")
	if err != nil {
		return nil, errors.Wrap(err, "error updating temple tree")
	}
	err = healingTempleLocation.AddInteraction(&ChangeLocationInteraction{
		Command: "leave the temple",
		Did:     homeLocation.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to temple tree")
	}

	victoryRoomTree, err := n.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating defeated dragon tree")
	}
	victoryRoomLocation := NewLocationTree(n, victoryRoomTree)
	err = victoryRoomLocation.SetDescription("You have defeated the dragon, his head lays on floor as prize for your victory.")
	if err != nil {
		return nil, errors.Wrap(err, "error updating defeated dragon tree")
	}
	err = victoryRoomLocation.AddInteraction(&ChangeLocationInteraction{
		Command: "return to Hogwarts' main hall",
		Did:     homeLocation.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interaction to defeated dragon tree")
	}

	dragonTree, err := n.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating dragon tree")
	}
  dragonObj := NewObjectTree(n, dragonTree)
  err = dragonObj.SetName("giant-dragon")
  if err != nil {
		return nil, errors.Wrap(err, "error setting name of new object")
  }
  dragonFightInteraction, err := NewCipherInteraction(
		"fight the dragon", "with courage",
		&ChangeLocationInteraction{
			Did: victoryRoomLocation.MustId(),
		},
		&ChangeLocationInteraction{
			Did: healingTempleLocation.MustId(),
		},
	)
	err = dragonObj.AddInteraction(dragonFightInteraction)
	if err != nil {
		return nil, errors.Wrap(err, "cipher tree error")
	}
  err = headmastersChambersInventory.Add(dragonObj.MustId())
  if err != nil {
		return nil, errors.Wrap(err, "error adding dragon")
	}

	dragonsHeadTree, err := n.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating dragonsHead tree")
	}
  dragonsHeadObj := NewObjectTree(n, dragonsHeadTree)
  err = dragonsHeadObj.SetName("dragons-head")
  if err != nil {
		return nil, errors.Wrap(err, "error setting name of new object")
	}
  err = dragonsHeadObj.SetDescription("this dragon head displays your great courage and victory in battle")
  if err != nil {
		return nil, errors.Wrap(err, "error setting description of new object")
	}
	err = dragonsHeadObj.AddInteraction(&GetTreeValueInteraction{
		Command: "examine dragons-head",
		Did:     dragonsHeadObj.MustId(),
		Path:    "description",
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interactions to new object")
	}
	err = dragonsHeadObj.AddInteraction(&PickUpObjectInteraction{
		Command: "pick up dragons-head",
		Did:     dragonsHeadObj.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interactions to new object")
	}

  victoryLocationInventory := trees.NewInventoryTree(n, victoryRoomTree)
  err = victoryLocationInventory.Add(dragonsHeadTree.MustId())
  if err != nil {
		return nil, errors.Wrap(err, "error adding dragons head")
	}

	return homeLocation, nil
}
