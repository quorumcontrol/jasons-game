package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	logging "github.com/ipfs/go-log"

	"github.com/ethereum/go-ethereum/crypto"
	s3ds "github.com/ipfs/go-ds-s3"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/jason/blockstore"
	"github.com/quorumcontrol/jasons-game/server"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
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

	var config s3ds.Config
	if *isLocal {
		// first sleep to give the localstack docker container time to startup
		time.Sleep(2 * time.Second)
		config = s3ds.Config{
			RegionEndpoint: "http://localstack:4572",
			Bucket:         localBucketName,
			Region:         "us-east-1",
			AccessKey:      "localonlyac",
			SecretKey:      "localonlysk",
		}
	} else {
		var bucket string
		var region string

		bucket, ok := os.LookupEnv("JASON_S3_BUCKET")
		if !ok {
			panic(fmt.Errorf("${JASON_S3_BUCKET} is required in non-local mode"))
		}

		region, ok = os.LookupEnv("AWS_REGION")
		if !ok {
			panic(fmt.Errorf("${AWS_REGION} is required in non-local mode"))
		}

		// we expect credentials to come from the normal ec2 environment
		config = s3ds.Config{
			Bucket: bucket,
			Region: region,
		}
	}

	ds, err := s3ds.NewS3Datastore(config)
	if err != nil {
		panic(errors.Wrap(err, "error creating store"))
	}

	if *isLocal {
		// because we're local, go ahead and create the bucket
		err := devMakeBucket(ds.S3, localBucketName)
		if err != nil {
			panic(fmt.Errorf("error creating bucket: %v", err))
		}
	}

	notaryGroup, err := server.SetupTupeloNotaryGroup(ctx, *isLocal)
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

func devMakeBucket(s3obj *s3.S3, bucketName string) error {
	_, err := s3obj.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})

	return err
}

func mustSetLogLevel(name, level string) {
	if err := logging.SetLogLevel(name, level); err != nil {
		panic(errors.Wrapf(err, "error setting log level of %s to %s", name, level))
	}
}
