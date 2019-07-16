// +build !public

package depositor

import (
	"context"

	"github.com/quorumcontrol/messages/build/go/transactions"

	"github.com/quorumcontrol/jasons-game/inkwell/config"
	"github.com/quorumcontrol/jasons-game/inkwell/ink"
)

type InkDepositor struct {
	iwconfig *config.Inkwell
	inkwell  ink.Well
}

func New(ctx context.Context, cfg config.InkwellConfig) (*InkDepositor, error) {
	iwconfig, err := config.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	cticonfig := ink.ChainTreeInkwellConfig{
		Net:         iwconfig.Net,
		InkOwnerDID: cfg.InkOwnerDID,
	}

	iw, err := ink.NewChainTreeInkwell(cticonfig)
	if err != nil {
		return nil, err
	}

	return &InkDepositor{iwconfig: iwconfig, inkwell: iw}, nil
}

func (id *InkDepositor) Deposit(tokenPayload *transactions.TokenPayload) error {
	return id.inkwell.DepositInk(tokenPayload)
}
