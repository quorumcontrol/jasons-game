// +build internal

package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/devink/devink"
)

const (
	dataStoreDir  = "dev-ink"
	minimumAmount = 1000000
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	local := flag.Bool("local", true, "connect to locally running notary group.")
	flag.Parse()

	devInkSource, err := devink.NewSource(ctx, dataStoreDir, local)
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

	err = devInkSource.EnsureBalance(ctx, minimumAmount)
	if err != nil {
		panic(err)
	}

	if len(os.Args) > 1 && len(os.Args[1]) > 0 {
		var destinationChainId string

		if strings.HasPrefix(os.Args[1], "did:tupelo:") {
			destinationChainId = os.Args[1]
		} else {
			// assume it's a base64-encoded private key
			serializedKey, err := base64.StdEncoding.DecodeString(os.Args[1])
			if err != nil {
				panic(errors.Wrap(err, "error decoding base64 ink faucet key argument"))
			}

			key, err := crypto.ToECDSA(serializedKey)
			if err != nil {
				panic(errors.Wrap(err, "error deserializing ink faucet key argument"))
			}

			destinationChainId = consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
		}

		tokenSend, err := devInkSource.SendInk(ctx, destinationChainId, minimumAmount)
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
