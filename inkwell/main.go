package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

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

	inkwell, err := server.NewServer(ctx, serverCfg)
	if err != nil {
		panic(errors.Wrap(err, "error creating new inkwell server"))
	}

	// TODO: Call some kind of inkwell.Start() to make it listen for player invites on a topic
	fmt.Printf("Started inkwell %+v\n", inkwell)

	<-make(chan struct{})
}
