package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/messages/build/go/transactions"

	"github.com/quorumcontrol/jasons-game/inkfaucet/config"
	"github.com/quorumcontrol/jasons-game/inkfaucet/depositor"
	"github.com/quorumcontrol/jasons-game/inkfaucet/ink"
	"github.com/quorumcontrol/jasons-game/inkfaucet/server"
)

const localBucketName = "tupelo-inkfaucet-local"

func mustSetLogLevel(name, level string) {
	if err := logging.SetLogLevel(name, level); err != nil {
		panic(errors.Wrapf(err, "error setting log level of %s to %s", name, level))
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mustSetLogLevel("*", "warning")
	mustSetLogLevel("pubsub", "error")
	mustSetLogLevel("inkfaucet", "debug")

	local := flag.Bool("local", false, "connect to localnet & use localstack S3 instead of testnet & real S3")
	outputdid := flag.Bool("outputdid", false, "output inkfaucet DID and exit")
	deposit := flag.String("deposit", "", "token payload for ink deposit")
	flag.Parse()

	var s3Region, s3Bucket string

	if *local {
		s3Bucket = localBucketName
	} else {
		var ok bool

		s3Bucket, ok = os.LookupEnv("INK_FAUCET_S3_BUCKET")
		if !ok {
			panic(fmt.Errorf("${INK_FAUCET_S3_BUCKET} is required in non-local mode"))
		}

		s3Region, ok = os.LookupEnv("AWS_REGION")
		if !ok {
			panic(fmt.Errorf("${AWS_REGION} is required in non-local mode"))
		}
	}

	inkDID := os.Getenv("INK_DID")

	inkfaucetCfg := config.InkFaucetConfig{
		Local:       *local,
		S3Region:    s3Region,
		S3Bucket:    s3Bucket,
		InkOwnerDID: inkDID,
	}

	if outputdid != nil && *outputdid {
		fmt.Println("Outputting inkfaucet DID")
		ctx := context.Background()
		iw, err := config.Setup(ctx, inkfaucetCfg)
		if err != nil {
			panic(err)
		}

		ct, err := iw.Net.GetChainTreeByName(ink.InkFaucetChainTreeName)
		if err != nil {
			panic(err)
		}

		fmt.Printf("INK_FAUCET_DID=%s\n", ct.MustId())

		os.Exit(0)
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
			panic(errors.Wrap(err, "error creating ink depositer"))
		}

		err = dep.Deposit(tokenPayload)
		if err != nil {
			panic(errors.Wrap(err, "error depositing ink"))
		}

		fmt.Println("Deposited ink into inkfaucet")

		os.Exit(0)
	}

	inkfaucet, err := server.New(ctx, inkfaucetCfg)
	if err != nil {
		panic(errors.Wrap(err, "error creating new inkfaucet server"))
	}

	err = inkfaucet.Start()
	if err != nil {
		panic(errors.Wrap(err, "error starting inkfaucet service"))
	}

	<-make(chan struct{})
}
