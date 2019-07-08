package depositer

import (
	"context"

	"github.com/quorumcontrol/messages/build/go/transactions"

	"github.com/quorumcontrol/jasons-game/inkwell/config"
	"github.com/quorumcontrol/jasons-game/inkwell/ink"
)

// TODO: Put this into a build tag that doesn't normally get included

type InkDepositer struct {
	iw *config.Inkwell
	is ink.Well
}

func New(ctx context.Context, cfg config.InkwellConfig) (*InkDepositer, error)  {
	iw, err := config.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	is, err := ink.NewChainTreeInkwell(ink.ChainTreeInkwellConfig{Net: iw.Net})
	if err != nil {
		return nil, err
	}

	return &InkDepositer{iw: iw, is: is}, nil
}

func (id *InkDepositer) Deposit(tokenPayload *transactions.TokenPayload) error {
	return id.is.DepositInk(tokenPayload)
}
