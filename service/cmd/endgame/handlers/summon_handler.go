package endgame 

import (
	"context"
	"fmt"
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	broadcastHandlers "github.com/quorumcontrol/jasons-game/handlers/broadcast"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

type EndGameSummonHandler struct {
	network network.Network
	did     string
	altars  []*EndGameAltar
}

var EndGameSummonHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName((*jasonsgame.RequestObjectTransferMessage)(nil)),
}

func NewEndGameSummonHandler(network network.Network, did string, altars []*EndGameAltar) handlers.Handler {
	h := &EndGameSummonHandler{
		network: network,
		did:     did,
		altars:  altars,
	}
	err := h.spawnNewPrize()
	if err != nil {
		log.Error(err)
	}
	return h
}

func (h *EndGameSummonHandler) Handle(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.RequestObjectTransferMessage:
		err := h.handleRequestObjectTransferMessage(msg)
		if err != nil {
			log.Error(err)
		}
		return err
	default:
		return handlers.ErrUnsupportedMessageType
	}
}

func (h *EndGameSummonHandler) handleRequestObjectTransferMessage(msg *jasonsgame.RequestObjectTransferMessage) error {
	if msg.From != h.did {
		return fmt.Errorf("wrong location did")
	}

	playerDid := msg.To
	playerTree, err := h.network.GetTree(playerDid)
	if err != nil {
		return fmt.Errorf("error fetching player tree: %v", err)
	}

	playerAuths, err := playerTree.Authentications()
	if err != nil {
		return fmt.Errorf("error fetching player auths: %v", err)
	}

	thisPlayerObjectsByAltar := make(map[string]*consensus.SignedChainTree)

	// Check win
	for _, altar := range h.altars {
		altarInventory, err := trees.FindInventoryTree(h.network, altar.Did)
		if err != nil {
			return fmt.Errorf("error fetching altar inventory chaintree: %v", err)
		}

		altarObjects, err := altarInventory.All()
		if err != nil {
			return fmt.Errorf("error fetching altar objects %s: %v", msg.Object, err)
		}

		for obj := range altarObjects {
			objectTree, err := h.network.GetTree(obj)
			if err != nil {
				return fmt.Errorf("error fetching object chaintree %s: %v", objectTree.MustId(), err)
			}

			sacrificerDidUncast, remaining, err := objectTree.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "sacrificed-by"})
			if len(remaining) > 0 || err != nil {
				return fmt.Errorf("error fetching previous owner for %s: %v", objectTree.MustId(), err)
			}

			sacrificerDid, ok := sacrificerDidUncast.(string)
			if !ok {
				return fmt.Errorf("error casting previous owner for %s", objectTree.MustId())
			}

			// Sacrafice was not for this player, skip
			if playerDid != sacrificerDid {
				continue
			}
			thisPlayerObjectsByAltar[altar.Did] = objectTree
		}
	}

	// check if players objects meet the winning criteria
	didWin, err := h.hasWinningObjects(thisPlayerObjectsByAltar)
	if err != nil {
		return fmt.Errorf("error on checking win status: %v", err)
	}

	objectTree, err := h.network.GetTree(msg.Object)
	if err != nil {
		return fmt.Errorf("error fetching object chaintree %s: %v", objectTree.MustId(), err)
	}

	var transferMessage string
	if didWin {
		// TODO: process all object changes in one PlayTransactions
		objectTree, err = h.network.UpdateChainTree(objectTree, "jasons-game/description", fmt.Sprintf("Congrats %v you have won satoshis treasure on tupelo", playerDid))
		if err != nil {
			return fmt.Errorf("error updating object chaintree: %v", err)
		}
		objectTree, err = h.network.UpdateChainTree(objectTree, "jasons-game/name", "satoshis-prize")
		if err != nil {
			return fmt.Errorf("error updating object chaintree: %v", err)
		}
		transferMessage = "Congrats, you have won"
	} else {
		// TODO: process all object changes in one PlayTransactions
		objectTree, err = h.network.UpdateChainTree(objectTree, "jasons-game/description", "better try next time")
		if err != nil {
			return fmt.Errorf("error updating object chaintree: %v", err)
		}
		objectTree, err = h.network.UpdateChainTree(objectTree, "jasons-game/name", "failure-note")
		if err != nil {
			return fmt.Errorf("error updating object chaintree: %v", err)
		}
		transferMessage = "Sorry you have failed"
	}

	playerInventory, err := trees.FindInventoryTree(h.network, playerDid)
	if err != nil {
		return fmt.Errorf("error fetching player inventory chaintree: %v", err)
	}

	remoteTargetHandler, err := handlers.FindHandlerForTree(h.network, playerDid)
	if err != nil {
		return fmt.Errorf("error fetching handler for %v", playerDid)
	}
	var targetHandler handlers.Handler
	if remoteTargetHandler != nil {
		targetHandler = remoteTargetHandler
	} else {
		targetHandler = broadcastHandlers.NewTopicBroadcastHandler(h.network, playerInventory.BroadcastTopic())
	}

	objectTree, err = h.network.ChangeChainTreeOwner(objectTree, playerAuths)
	if err != nil {
		return fmt.Errorf("error updating object chaintree: %v", err)
	}

	transferredObjectMessage := &jasonsgame.TransferredObjectMessage{
		From:    h.did,
		To:      playerDid,
		Object:  objectTree.MustId(),
		Message: transferMessage,
	}

	err = targetHandler.Handle(transferredObjectMessage)
	if err != nil {
		return err
	}

	err = h.spawnNewPrize()
	if err != nil {
		log.Error(err)
	}

	// Trash players object regardless if they won or lost
	for altarDid, obj := range thisPlayerObjectsByAltar {
		altarInventory, err := trees.FindInventoryTree(h.network, altarDid)
		if err != nil {
			return fmt.Errorf("error fetching altar inventory chaintree: %v", err)
		}
		err = altarInventory.Remove(obj.MustId())
		if err != nil {
			return fmt.Errorf("error deleting object from altar inventory chaintree: %v", err)
		}
		// TODO, destroy object in some better way, 0 out, remove auths?
	}

	return nil
}

func (h *EndGameSummonHandler) hasWinningObjects(toVerify map[string]*consensus.SignedChainTree) (bool, error) {
	// Clearly wrong, some altars are empty
	if len(toVerify) != len(h.altars) {
		return false, nil
	}

	for _, altar := range h.altars {
		sortedRequires := make([]string, len(altar.Requires))
		copy(sortedRequires, altar.Requires)
		sort.Strings(sortedRequires)

		objectTree := toVerify[altar.Did]

		inscriptionsUncast, remaining, err := objectTree.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "inscriptions"})
		if err != nil {
			return false, fmt.Errorf("error fetching inscriptions on %s: %v", objectTree.MustId(), err)
		}

		// incorrect: no inscriptions on this object
		if len(remaining) > 0 {
			return false, nil
		}

		inscriptionsSliceUncast, ok := inscriptionsUncast.([]interface{})
		if !ok {
			return false, fmt.Errorf("error casting inscriptions on %s: %v", objectTree.MustId(), err)
		}

		// incorrect: inscriptions length is different than expected length
		if len(inscriptionsSliceUncast) != len(altar.Requires) {
			return false, nil
		}

		inscriptions := make([]string, len(inscriptionsSliceUncast))
		for i, v := range inscriptionsSliceUncast {
			inscriptions[i] = v.(string)
		}
		sort.Strings(inscriptions)

		for i, val := range sortedRequires {
			if val != inscriptions[i] {
				// incorrect: inscriptions length is different than expected length
				return false, nil
			}
		}
	}

	// correct: all altars have checked toVerify objects and agreed
	return true, nil
}

func (h *EndGameSummonHandler) spawnNewPrize() error {
	prizeTree, err := h.network.CreateChainTree()
	if err != nil {
		return fmt.Errorf("error creating prize tree %s: %v", prizeTree.MustId(), err)
	}

	prizeObj := game.NewObjectTree(h.network, prizeTree)
	name := "the-judgement-throne"

	err = prizeObj.SetName(name)
	if err != nil {
		return errors.Wrap(err, "error setting name of new object")
	}

	inventory, err := trees.FindInventoryTree(h.network, h.did)
	if err != nil {
		return fmt.Errorf("error fetching summon inventory chaintree: %v", err)
	}

	// TODO: Do all theses changes in one PlayTransaction
	objects, err := inventory.All()
	if err != nil {
		return fmt.Errorf("error fetching objects: %v", err)
	}

	for objDid, objName := range objects {
		if objName == name {
			err = inventory.Remove(objDid)
			if err != nil {
				log.Error(err)
			}
		}
	}

	err = inventory.Add(prizeObj.MustId())
	if err != nil {
		return fmt.Errorf("error adding prize to inventory chaintree: %v", err)
	}

	locationTree, err := h.network.GetTree(h.did)
	if err != nil {
		return fmt.Errorf("error fetching player tree: %v", err)
	}

	summonInteraction := &game.PickUpObjectInteraction{
		Command: "summon the judgement",
		Did:     prizeTree.MustId(),
	}

	location := game.NewLocationTree(h.network, locationTree)
	err = location.RemoveInteraction(summonInteraction.Command)
	if err != nil {
		log.Error(err)
	}

	err = location.AddInteraction(summonInteraction)
	if err != nil {
		return fmt.Errorf("error adding interaction to location tree: %v", err)
	}

	return nil
}

func (h *EndGameSummonHandler) Supports(msg proto.Message) bool {
	return EndGameSummonHandlerMessages.Contains(msg)
}

func (h *EndGameSummonHandler) SupportedMessages() []string {
	return EndGameSummonHandlerMessages
}
