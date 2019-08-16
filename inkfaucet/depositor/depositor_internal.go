// +build internal

package depositor

import (
	"context"
	"fmt"

	"github.com/quorumcontrol/messages/build/go/transactions"

	"github.com/quorumcontrol/jasons-game/inkfaucet/config"
	"github.com/quorumcontrol/jasons-game/inkfaucet/ink"
	"github.com/quorumcontrol/jasons-game/network"
)

type InkDepositor struct {
	ifconfig  *config.InkFaucet
	inkfaucet ink.Faucet
	net       network.Network
}

func New(ctx context.Context, cfg config.InkFaucetConfig) (*InkDepositor, error) {
	ifconfig, err := config.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	cticonfig := ink.ChainTreeInkFaucetConfig{
		Net:         ifconfig.Net,
		InkOwnerDID: cfg.InkOwnerDID,
		PrivateKey:  cfg.PrivateKey,
	}

	inkFaucet, err := ink.NewChainTreeInkFaucet(cticonfig)
	if err != nil {
		return nil, err
	}

	return &InkDepositor{ifconfig: ifconfig, inkfaucet: inkFaucet, net: ifconfig.Net}, nil
}

func (id *InkDepositor) Deposit(tokenPayload *transactions.TokenPayload) error {
	err := id.inkfaucet.DepositInk(tokenPayload)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	_, err = id.net.GetTree(id.inkfaucet.ChainTreeDID())
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	return err
}
