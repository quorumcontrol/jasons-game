package arcadia

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/courts/artifact"
	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/utils/stringslice"
)

type EndGamePrizeHandler struct {
	*court.PrizeHandler
	net          network.Network
	court        *ArcadiaCourt
	altarDids    []string
	artifactsCfg *artifact.ArtifactsConfig
}

func NewEndGamePrizeHandler(c *ArcadiaCourt) (*EndGamePrizeHandler, error) {
	handler := &EndGamePrizeHandler{
		net:       c.net,
		court:     c,
		altarDids: c.altarDids,
	}
	var err error
	handler.PrizeHandler, err = court.NewPrizeHandler(&court.PrizeHandlerConfig{
		Court:           c.court,
		PrizeConfigPath: filepath.Join(c.configPath, "arcadia/prize_config.yml"),
		ValidatorFunc:   handler.validatorFunc,
		CleanupFunc:     handler.cleanupFunc,
	})
	if err != nil {
		return nil, err
	}

	handler.artifactsCfg, err = artifact.NewArtifactsConfig(filepath.Join(c.configPath, "artifacts"))
	if err != nil {
		return nil, err
	}

	return handler, nil
}

func (h *EndGamePrizeHandler) cleanupFunc(msg *jasonsgame.RequestObjectTransferMessage) error {
	playerTree, err := h.net.GetTree(msg.To)
	if err != nil {
		return errors.Wrap(err, "error fetching player tree")
	}

	playerInventoriesMap, _, err := playerTree.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "location-inventories"})
	if err != nil || playerInventoriesMap == nil {
		return fmt.Errorf("could not resolve player location inventories")
	}

	// build a map with expected altar location dids to player-location inventory dids
	playerAltarInventories := make(map[string]string)
	for _, altarDid := range h.altarDids {
		playerAltarInventories[altarDid] = ""
	}

	for locationDid, inventoryDidUncast := range playerInventoriesMap.(map[string]interface{}) {
		if _, ok := playerAltarInventories[locationDid]; ok {
			playerAltarInventories[locationDid] = inventoryDidUncast.(string)
		}
	}

	for _, invDid := range playerAltarInventories {
		if invDid == "" {
			continue
		}

		inventory, err := trees.FindInventoryTree(h.net, invDid)
		if err != nil {
			continue
		}

		all, err := inventory.All()
		if err != nil {
			continue
		}

		for itemDid, itemName := range all {
			if strings.HasPrefix(itemName, "artifact-") {
				err = inventory.Remove(itemDid)
				if err != nil {
					continue
				}
			}
		}
	}

	return nil
}

func (h *EndGamePrizeHandler) validatorFunc(msg *jasonsgame.RequestObjectTransferMessage) (bool, error) {
	ctx := context.Background()

	log.Infof("endgame validator: starting: player=%s obj=%s", msg.To, msg.Object)

	playerTree, err := h.net.GetTree(msg.To)
	if err != nil {
		return false, errors.Wrap(err, "error fetching player tree")
	}

	playerInventoriesMap, _, err := playerTree.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "location-inventories"})
	if err != nil || playerInventoriesMap == nil {
		return false, fmt.Errorf("error fetching player inventory chaintree: %v", err)
	}

	// build a map with expected altar location dids to player-location inventory dids
	playerAltarInventories := make(map[string]string)
	for _, altarDid := range h.altarDids {
		playerAltarInventories[altarDid] = ""
	}

	for locationDid, inventoryDidUncast := range playerInventoriesMap.(map[string]interface{}) {
		if _, ok := playerAltarInventories[locationDid]; ok {
			playerAltarInventories[locationDid] = inventoryDidUncast.(string)
		}
	}

	log.Debugf("endgame validator: altar inventories map: player=%s inventories=%v", msg.To, playerAltarInventories)

	altarsToArtifacts := make(map[string][]string)

	for altarDid, invDid := range playerAltarInventories {
		if invDid == "" {
			return false, fmt.Errorf("you must place one artifact on each altar")
		}

		inventory, err := trees.FindInventoryTree(h.net, invDid)
		if err != nil {
			return false, err
		}

		all, err := inventory.All()
		if err != nil {
			return false, err
		}

		log.Debugf("endgame validator: altar inventory: altar=%s objects=%s", invDid, all)

		artifacts := []string{}

		for itemDid, itemName := range all {
			if strings.HasPrefix(itemName, "artifact-") {
				artifacts = append(artifacts, itemDid)
			}
		}

		altarsToArtifacts[altarDid] = artifacts
	}

	log.Debugf("endgame validator: altar to artifacts: player=%s artifacts=%s", msg.To, altarsToArtifacts)

	if len(altarsToArtifacts) < len(h.altarDids) {
		return false, fmt.Errorf("you must place one artifact on each altar")
	}

	validCount := 0

	for altarIndex, altarDid := range h.altarDids {
		artifacts := altarsToArtifacts[altarDid]

		// can only have one artifact per altar, fail
		if len(artifacts) != 1 {
			break
		}

		artifact, err := h.net.GetTree(artifacts[0])
		if err != nil {
			return false, err
		}

		ownershipChanges, err := trees.OwnershipChanges(ctx, artifact.ChainTree)
		if err != nil {
			return false, errors.Wrap(err, "error checking origin of artifact")
		}

		if len(ownershipChanges) < 2 {
			return false, errors.Wrap(err, "invalid ownership history")
		}

		expectedOriginAuths := []string{h.artifactsCfg.Artifacts[altarIndex].OriginAuth}

		atCreationOwnership := ownershipChanges[len(ownershipChanges)-1]
		if !stringslice.Equal(atCreationOwnership.Authentications, expectedOriginAuths) {
			log.Debugf("endgame validator: incorrect creation ownership: artifact=%s expectedAuth=%v foundAuth=%v", artifact.MustId(), expectedOriginAuths, atCreationOwnership.Authentications)
			break
		}

		expectedEmptyArtifact, err := h.net.GetTreeByTip(atCreationOwnership.Tip)
		if err != nil {
			return false, fmt.Errorf("error checking origin of element")
		}
		expectedEmptyData, _, err := expectedEmptyArtifact.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data"})
		if err != nil {
			return false, fmt.Errorf("error checking origin of element")
		}

		if expectedEmptyData != nil {
			log.Debugf("endgame validator: artifact at creation was not empty: artifact=%s", artifact.MustId())
			break
		}

		beforeTransferOwnership := ownershipChanges[len(ownershipChanges)-2]
		originArtifact, err := h.net.GetTreeByTip(beforeTransferOwnership.Tip)
		if err != nil {
			return false, fmt.Errorf("error checking origin of element")
		}
		validOrigin, err := trees.VerifyOwnership(ctx, originArtifact.ChainTree, expectedOriginAuths)
		if err != nil {
			return false, fmt.Errorf("error checking origin of element")
		}
		if !validOrigin {
			log.Debugf("endgame validator: origin auths were incorrect: artifact=%s", artifact.MustId())
			break
		}
		originInscriptionsUncast, _, err := originArtifact.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "inscriptions"})
		if err != nil {
			return false, err
		}
		originInscriptions, ok := originInscriptionsUncast.(map[string]interface{})
		if !ok {
			break
		}

		currentInscriptionsUncast, _, err := artifact.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "inscriptions"})
		if err != nil {
			return false, err
		}

		currentInscriptions, ok := currentInscriptionsUncast.(map[string]interface{})
		if !ok {
			break
		}

		expectedInscriptions := h.artifactsCfg.Artifacts[altarIndex].Inscriptions

		log.Debugf("endgame validator: inscription check for: artifact=%s altar=%d current=%v origin=%v", artifact.MustId(), altarIndex, currentInscriptions, originInscriptions)

		correctInscriptions := 0

		for inscriptionKey, inscriptionValUncast := range currentInscriptions {
			inscriptionVal, ok := inscriptionValUncast.(string)
			if !ok {
				break
			}

			correct := false

			switch inscriptionKey {
			case "type":
				correct = expectedInscriptions.Type == inscriptionVal
			case "material":
				correct = expectedInscriptions.Material == inscriptionVal
			case "age":
				correct = expectedInscriptions.Age == inscriptionVal
			case "weight":
				correct = expectedInscriptions.Weight == inscriptionVal
			case "forged by":
				originVal, ok := originInscriptions[inscriptionKey].(string)
				if !ok {
					break
				}
				correct = originVal == inscriptionVal && expectedInscriptions.ForgedBy == inscriptionVal
			}

			if !correct {
				log.Debugf("endgame validator: inscription check failed: artifact=%s key=%s", artifact.MustId(), inscriptionKey)
				break
			}

			correctInscriptions++
		}

		if correctInscriptions == 5 {
			validCount++
		}
	}

	if validCount == len(h.altarDids) {
		log.Infof("endgame validator: winner: player=%s obj=%s", msg.To, msg.Object)
		return true, nil
	}

	// Failure, delete artifacts
	for altarDid, artifacts := range altarsToArtifacts {
		invDid, ok := playerAltarInventories[altarDid]
		if !ok || invDid == "" {
			log.Errorf("altar inventory for player not found")
			continue
		}

		inventory, err := trees.FindInventoryTree(h.net, invDid)
		if err != nil {
			log.Errorf("could not find inventory tree")
			continue
		}

		for _, artifactDid := range artifacts {
			err = inventory.Remove(artifactDid)
			if err != nil {
				log.Errorf("could not remove bad artifact from inventory")
				continue
			}
		}
	}

	log.Infof("endgame validator: failed: obj=%s correctCount=%d", msg.Object, validCount)
	return false, fmt.Errorf("your altar placement is incorrect")
}
