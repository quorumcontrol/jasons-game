package autumn

import (
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

type AutumnPrizeHandler struct {
	*court.PrizeHandler
	net   network.Network
	court *AutumnCourt
}

func NewAutumnPrizeHandler(c *AutumnCourt) (*AutumnPrizeHandler, error) {
	handler := &AutumnPrizeHandler{
		net:   c.net,
		court: c,
	}
	var err error
	handler.PrizeHandler, err = court.NewPrizeHandler(&court.PrizeHandlerConfig{
		Court:           c.court,
		PrizeConfigPath: filepath.Join(c.configPath, "autumn/prize_config.yml"),
		ValidatorFunc:   handler.validatorFunc,
	})
	if err != nil {
		return nil, err
	}
	return handler, nil
}

func (h *AutumnPrizeHandler) validatorFunc(msg *jasonsgame.RequestObjectTransferMessage) (bool, error) {
	failMsg := h.court.config.PrizeFailMsg
	winningElementName := (&element{ID: h.court.config.WinningElement}).Name()

	playerInventory, err := trees.FindInventoryTree(h.net, msg.To)
	if err != nil {
		log.Error(err)
		return false, fmt.Errorf("could not fetch player inventory")
	}

	elementDid, err := playerInventory.DidForName(winningElementName)
	if err != nil {
		log.Error(err)
		return false, fmt.Errorf("could not fetch player inventory")
	}

	if elementDid == "" {
		return false, fmt.Errorf(failMsg)
	}

	elementObj, err := game.FindObjectTree(h.net, elementDid)
	if err != nil {
		log.Error(err)
		return false, fmt.Errorf("could not fetch element")
	}

	isValid, err := validateElementOrigin(elementObj, []string{crypto.PubkeyToAddress(*h.net.PublicKey()).String()})
	if err != nil {
		log.Error(err)
		return false, fmt.Errorf(failMsg)
	}

	if !isValid {
		return false, fmt.Errorf(failMsg)
	}

	return true, nil
}
