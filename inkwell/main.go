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

	"github.com/quorumcontrol/jasons-game/inkwell/server"
)

const localBucketName = "tupelo-inkwell-local"

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
	mustSetLogLevel("inkwell", "debug")

	local := flag.Bool("local", false, "connect to localnet & use localstack S3 instead of testnet & real S3")
	deposit := flag.String("deposit", "", "token payload for ink deposit")
	flag.Parse()

	var s3Region, s3Bucket string

    if *local {
    	s3Bucket = localBucketName
	} else {
		var ok bool

		s3Bucket, ok = os.LookupEnv("INKWELL_S3_BUCKET")
		if !ok {
			panic(fmt.Errorf("${INKWELL_S3_BUCKET} is required in non-local mode"))
		}

		s3Region, ok = os.LookupEnv("AWS_REGION")
		if !ok {
			panic(fmt.Errorf("${AWS_REGION} is required in non-local mode"))
		}
	}

	serverCfg := server.InkwellConfig{
		Local:    *local,
		S3Region: s3Region,
		S3Bucket: s3Bucket,
	}

	if deposit != nil && *deposit != "" {
		marshalledTokenPayload, err := base64.StdEncoding.DecodeString(*deposit)
		if err != nil {
			panic(fmt.Errorf("error base64 decoding ink deposit token payload: %v", err))
		}

		tokenPayload := transactions.TokenPayload{}
		err = proto.Unmarshal(marshalledTokenPayload, &tokenPayload)
		if err != nil {
			panic(fmt.Errorf("error unmarshalling ink deposit token payload: %v", err))
		}



		return
	}



	inkwell, err := server.NewServer(ctx, serverCfg)
	if err != nil {
		panic(errors.Wrap(err, "error creating new inkwell server"))
	}

	err = inkwell.Start()
	if err != nil {
		panic(errors.Wrap(err, "error starting inkwell service"))
	}

	<-make(chan struct{})
}
