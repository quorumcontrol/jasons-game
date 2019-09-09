package spring

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

const pedestalCount = 3

type SpringPrizeHandler struct {
	*court.PrizeHandler
	net   network.Network
	court *SpringCourt
}

func NewSpringPrizeHandler(c *SpringCourt) (*SpringPrizeHandler, error) {
	handler := &SpringPrizeHandler{
		net:   c.net,
		court: c,
	}
	var err error
	handler.PrizeHandler, err = court.NewPrizeHandler(&court.PrizeHandlerConfig{
		Court:           c.court,
		PrizeConfigPath: filepath.Join(c.configPath, "spring/prize_config.yml"),
		ValidatorFunc:   handler.validatorFunc,
		CleanupFunc:     handler.cleanupFunc,
	})
	if err != nil {
		return nil, err
	}
	return handler, nil
}

func (h *SpringPrizeHandler) cleanupFunc(msg *jasonsgame.RequestObjectTransferMessage) error {
	playerTree, err := h.net.GetTree(msg.To)
	if err != nil {
		return errors.Wrap(err, "error fetching player tree")
	}

	playerInventoriesMap, _, err := playerTree.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "location-inventories"})
	if err != nil || playerInventoriesMap == nil {
		return fmt.Errorf("could not resolve player location inventories")
	}

	pedestalDids := h.court.config.Pedestals

	// build a map with expected pedestal location dids to player-location inventory dids
	playerPedestalInventories := make(map[string]string)
	for pedestalDid := range pedestalDids {
		playerPedestalInventories[pedestalDid] = ""
	}

	for locationDid, inventoryDidUncast := range playerInventoriesMap.(map[string]interface{}) {
		if _, ok := playerPedestalInventories[locationDid]; ok {
			playerPedestalInventories[locationDid] = inventoryDidUncast.(string)
		}
	}

	for _, invDid := range playerPedestalInventories {
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
			if strings.HasPrefix(itemName, "page-") {
				err = inventory.Remove(itemDid)
				if err != nil {
					continue
				}
			}
		}
	}

	return nil
}

func (h *SpringPrizeHandler) validatorFunc(msg *jasonsgame.RequestObjectTransferMessage) (bool, error) {
	playerTree, err := h.net.GetTree(msg.To)
	if err != nil {
		return false, errors.Wrap(err, "error fetching player tree")
	}

	playerInventoriesMap, _, err := playerTree.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "location-inventories"})
	if err != nil || playerInventoriesMap == nil {
		return false, fmt.Errorf("error fetching player inventory chaintree: %v", err)
	}

	pedestalDids := h.court.config.Pedestals

	// build a map with expected pedestal location dids to player-location inventory dids
	playerPedestalInventories := make(map[string]string)
	for pedestalDid := range pedestalDids {
		playerPedestalInventories[pedestalDid] = ""
	}

	for locationDid, inventoryDidUncast := range playerInventoriesMap.(map[string]interface{}) {
		if _, ok := playerPedestalInventories[locationDid]; ok {
			playerPedestalInventories[locationDid] = inventoryDidUncast.(string)
		}
	}

	pedestalsToPages := make(map[string][]string)

	for pedestalDid, invDid := range playerPedestalInventories {
		if invDid == "" {
			return false, fmt.Errorf("you must place one page on each pedestal")
		}

		inventory, err := trees.FindInventoryTree(h.net, invDid)
		if err != nil {
			return false, err
		}

		all, err := inventory.All()
		if err != nil {
			return false, err
		}

		pages := []string{}

		for itemDid, itemName := range all {
			if strings.HasPrefix(itemName, "page-") {
				pages = append(pages, itemDid)
			}
		}

		pedestalsToPages[pedestalDid] = pages
	}

	if len(pedestalsToPages) < pedestalCount {
		return false, fmt.Errorf("you must place one page on each pedestal")
	}

	validCount := 0

	for pedestalDid, expectedInscription := range pedestalDids {
		pages := pedestalsToPages[pedestalDid]

		for _, pageDid := range pages {
			page, err := h.net.GetTree(pageDid)
			if err != nil {
				return false, err
			}

			inscriptionsUncast, _, err := page.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "inscriptions"})
			if err != nil {
				return false, err
			}

			inscriptions, ok := inscriptionsUncast.([]interface{})
			if !ok {
				return false, err
			}

			if len(inscriptions) == 1 && inscriptions[0].(string) == expectedInscription {
				validCount++
				break
			}
		}
	}

	if validCount == pedestalCount {
		return true, nil
	}

	// Failure, delete pages
	for pedestalDid, pages := range pedestalsToPages {
		invDid, ok := playerPedestalInventories[pedestalDid]
		if !ok || invDid == "" {
			log.Errorf("pedestal inventory for player not found")
			continue
		}

		inventory, err := trees.FindInventoryTree(h.net, invDid)
		if err != nil {
			log.Errorf("could not find inventory tree")
			continue
		}

		for _, pageDid := range pages {
			err = inventory.Remove(pageDid)
			if err != nil {
				log.Errorf("could not remove bad page from inventory")
				continue
			}
		}
	}

	return false, fmt.Errorf("your pedestal placement is incorrect")
}
