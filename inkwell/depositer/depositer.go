package depositer

import (
	"context"

	"github.com/quorumcontrol/messages/build/go/transactions"

	"github.com/quorumcontrol/jasons-game/inkwell/config"
	"github.com/quorumcontrol/jasons-game/inkwell/ink"
)

// TODO: Put this into a build tag that doesn't normally get included

type InkDepositer struct {
	iwconfig *config.Inkwell
	inkwell  ink.Well
}

func New(ctx context.Context, cfg config.InkwellConfig) (*InkDepositer, error)  {
	iwconfig, err := config.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	iw, err := ink.NewChainTreeInkwell(ink.ChainTreeInkwellConfig{Net: iwconfig.Net})
	if err != nil {
		return nil, err
	}

	return &InkDepositer{iwconfig: iwconfig, inkwell: iw}, nil
}

func (id *InkDepositer) Deposit(tokenPayload *transactions.TokenPayload) error {
	return id.inkwell.DepositInk(tokenPayload)
}
