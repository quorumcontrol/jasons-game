// +build internal

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/devink/devink"
)

const (
	dataStoreDir = "dev-ink"
	amount       = 10000
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	devInkSource, err := devink.NewSource(ctx, dataStoreDir, true)
	if err != nil {
		panic(errors.Wrap(err, "error initializing dev ink"))
	}

	devInkChainTree := devInkSource.ChainTree

	devInkDID := devInkChainTree.MustId()

	fmt.Printf("INK_DID=%s\n", devInkDID)

	err = devInkSource.EnsureToken(ctx)
	if err != nil {
		panic(err)
	}

	err = devInkSource.EnsureBalance(ctx, amount)
	if err != nil {
		panic(err)
	}

	if len(os.Args) > 1 && len(os.Args[1]) > 0 {
		destinationChainId := os.Args[1]

		tokenSend, err := devInkSource.SendInk(ctx, destinationChainId, amount)
		if err != nil {
			panic(err)
		}

		serializedTokenSend, err := proto.Marshal(tokenSend)
		if err != nil {
			panic(errors.Wrap(err, "error serializing dev ink token payload"))
		}

		encodedTokenSend := base64.StdEncoding.EncodeToString(serializedTokenSend)

		fmt.Printf("TOKEN_PAYLOAD=%s\n", encodedTokenSend)
	}
}
