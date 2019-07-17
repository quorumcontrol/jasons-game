// +build !internal

package depositor

import (
	"context"
	"errors"

	"github.com/quorumcontrol/messages/build/go/transactions"

	"github.com/quorumcontrol/jasons-game/inkfaucet/config"
	"github.com/quorumcontrol/jasons-game/inkfaucet/ink"
)

// no depositor in public builds but the compiler wants the types & funcs to exist

type InkDepositor struct {
	iwconfig  *config.InkFaucet
	inkfaucet ink.Faucet
}

func New(ctx context.Context, cfg config.InkFaucetConfig) (*InkDepositor, error) {
	return nil, errors.New("unavailable in public build")
}

func (id *InkDepositor) Deposit(tokenPayload *transactions.TokenPayload) error {
	return errors.New("unavailable in public build")
}
