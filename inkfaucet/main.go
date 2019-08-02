package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/build"
	"github.com/quorumcontrol/jasons-game/inkfaucet/config"
	"github.com/quorumcontrol/jasons-game/inkfaucet/depositor"
	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
	"github.com/quorumcontrol/jasons-game/inkfaucet/server"
)

func mustSetLogLevel(name, level string) {
	if err := logging.SetLogLevel(name, level); err != nil {
		panic(errors.Wrapf(err, "error setting log level of %s to %s", name, level))
	}
}

func keyToDID(key *ecdsa.PrivateKey) string {
	return consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mustSetLogLevel("*", "warning")
	mustSetLogLevel("pubsub", "error")
	mustSetLogLevel("invites", "debug")
	mustSetLogLevel("inkFaucet", "debug")
	mustSetLogLevel("gamenetwork", "info")

	local := flag.Bool("local", false, "connect to localnet & use localstack S3 instead of testnet & real S3")
	deposit := flag.String("deposit", "", "token payload for ink deposit")

	var invite *bool
	if build.BuildLabel == "internal" {
		invite = flag.Bool("invite", false, "generate a new player invite code")
	}

	flag.Parse()

	inkDID := os.Getenv("INK_DID")
	inkFaucetEncodedKey := os.Getenv("INK_FAUCET_KEY")

	var (
		signingKey   *ecdsa.PrivateKey
		inkFaucetDID string
		err          error
	)

	if inkFaucetEncodedKey == "" {
		if deposit != nil && *deposit != "" {
			panic("INK_FAUCET_KEY must be set for deposits")
		}

		if invite != nil && *invite {
			panic("INK_FAUCET_KEY must be set for invites")
		}

		fmt.Println("Private key not set; generating a new one")
		signingKey, err = crypto.GenerateKey()
		if err != nil {
			panic(errors.Wrap(err, "error generating private key for ink faucet"))
		}

		inkFaucetDID = keyToDID(signingKey)

		fmt.Printf("INK_FAUCET_KEY=%s\n", base64.StdEncoding.EncodeToString(crypto.FromECDSA(signingKey)))
		fmt.Printf("INK_FAUCET_DID=%s\n", inkFaucetDID)

		os.Exit(0)
	}

	inkFaucetSerializedKey, err := base64.StdEncoding.DecodeString(inkFaucetEncodedKey)
	if err != nil {
		panic(errors.Wrap(err, "error decoding ink faucet key"))
	}

	signingKey, err = crypto.ToECDSA(inkFaucetSerializedKey)
	if err != nil {
		panic(errors.Wrap(err, "error unserializing ink faucet key"))
	}

	inkFaucetDID = keyToDID(signingKey)

	inkfaucetCfg := config.InkFaucetConfig{
		Local:        *local,
		InkOwnerDID:  inkDID,
		InkFaucetDID: inkFaucetDID,
		PrivateKey:   signingKey,
	}

	if deposit != nil && *deposit != "" {
		fmt.Println("Making a deposit")

		marshalledTokenPayload, err := base64.StdEncoding.DecodeString(*deposit)
		if err != nil {
			panic(errors.Wrap(err, "error base64 decoding ink deposit token payload"))
		}

		tokenPayload := &transactions.TokenPayload{}
		err = proto.Unmarshal(marshalledTokenPayload, tokenPayload)
		if err != nil {
			panic(errors.Wrap(err, "error unmarshalling ink deposit token payload"))
		}

		dep, err := depositor.New(ctx, inkfaucetCfg)
		if err != nil {
			panic(errors.Wrap(err, "error creating ink depositor"))
		}

		err = dep.Deposit(tokenPayload)
		if err != nil {
			panic(errors.Wrap(err, "error depositing ink"))
		}

		fmt.Println("Deposited ink into ink faucet")

		os.Exit(0)
	}

	inkFaucetRouter, err := server.New(ctx, inkfaucetCfg)
	if err != nil {
		panic(errors.Wrap(err, "error creating new inkFaucet server"))
	}

	err = inkFaucetRouter.Start(*invite)
	if err != nil {
		panic(errors.Wrap(err, "error starting inkFaucet service"))
	}

	if invite != nil && *invite {
		if build.BuildLabel != "internal" {
			os.Exit(1)
		}

		actorCtx := actor.EmptyRootContext

		inviteReq := &inkfaucet.InviteRequest{}
		inviteActorReq := actorCtx.RequestFuture(inkFaucetRouter.PID(), inviteReq, 35*time.Second)

		uncastReq, err := inviteActorReq.Result()
		if err != nil {
			panic(errors.Wrap(err, "error requesting invite"))
		}

		inviteResp, ok := uncastReq.(*inkfaucet.InviteResponse)
		if !ok {
			panic(errors.Errorf("error casting invite request of type %T", uncastReq))
		}

		fmt.Println("\n\ninvite code:", inviteResp.Invite)

		os.Exit(0)
	}

	<-make(chan struct{})
}
