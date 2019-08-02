// +build internal

package depositor

import (
	"context"

	"github.com/quorumcontrol/messages/build/go/transactions"

	"github.com/quorumcontrol/jasons-game/inkfaucet/config"
	"github.com/quorumcontrol/jasons-game/inkfaucet/ink"
)

type InkDepositor struct {
	iwconfig  *config.InkFaucet
	inkfaucet ink.Faucet
}

func New(ctx context.Context, cfg config.InkFaucetConfig) (*InkDepositor, error) {
	iwconfig, err := config.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	cticonfig := ink.ChainTreeInkFaucetConfig{
		Net:         iwconfig.Net,
		InkOwnerDID: cfg.InkOwnerDID,
	}

	iw, err := ink.NewChainTreeInkFaucet(cticonfig)
	if err != nil {
		return nil, err
	}

	return &InkDepositor{iwconfig: iwconfig, inkfaucet: iw}, nil
}

func (id *InkDepositor) Deposit(tokenPayload *transactions.TokenPayload) error {
	return id.inkfaucet.DepositInk(tokenPayload)
}
