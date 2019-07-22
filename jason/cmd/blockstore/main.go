package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	logging "github.com/ipfs/go-log"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/jason/blockstore"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/config"
)

const (
	localBucketName = "tupelo-jason-blocks-local"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	isLocal := flag.Bool("local", false, "turn on localmode config for s3")
	flag.Parse()

	mustSetLogLevel("*", "warning")
	mustSetLogLevel("pubsub", "error")
	mustSetLogLevel("jasonblocks", "debug")

	var bucket string
	var region string

	if *isLocal {
		bucket = localBucketName
	} else {
		var ok bool

		bucket, ok = os.LookupEnv("JASON_S3_BUCKET")
		if !ok {
			panic(fmt.Errorf("${JASON_S3_BUCKET} is required in non-local mode"))
		}

		region, ok = os.LookupEnv("AWS_REGION")
		if !ok {
			panic(fmt.Errorf("${AWS_REGION} is required in non-local mode"))
		}
	}

	ds, err := config.S3DataStore(*isLocal, region, bucket)
	if err != nil {
		panic(errors.Wrap(err, "error configuring S3 data store"))
	}

	ds, err = config.LocalDataStore("/tmp/importer")
	if err != nil {
		panic(errors.Wrap(err, "error configuring local data store"))
	}

	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, *isLocal)
	if err != nil {
		panic(errors.Wrap(err, "error setting up tupelo notary group"))
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		panic(errors.Wrap(err, "error generating node key"))
	}

	p, err := blockstore.New(ctx, key, ds, notaryGroup)
	if err != nil {
		panic(errors.Wrap(err, "error creating provider"))
	}
	err = p.Start()
	if err != nil {
		panic(errors.Wrap(err, "error starting provider"))
	}

	<-make(chan struct{})
}

func mustSetLogLevel(name, level string) {
	if err := logging.SetLogLevel(name, level); err != nil {
		panic(errors.Wrapf(err, "error setting log level of %s to %s", name, level))
	}
}
