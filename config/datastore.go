package config

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	s3ds "github.com/ipfs/go-ds-s3"
	"github.com/pkg/errors"
)

func LocalDataStore(path string) (datastore.Batching, error) {
	ds, err := badger.NewDatastore(path, &badger.DefaultOptions)
	if err != nil {
		return nil, errors.Wrap(err, "error creating store")
	}

	return ds, nil
}

func S3DataStore(isLocal bool, region, bucket string) (datastore.Batching, error) {
	var config s3ds.Config

	if isLocal {
		// first sleep to give the localstack docker container time to startup
		time.Sleep(2 * time.Second)
		config = s3ds.Config{
			RegionEndpoint: "http://localstack:4572",
			Bucket:         bucket,
			Region:         "us-east-1",
			AccessKey:      "localonlyac",
			SecretKey:      "localonlysk",
		}
	} else {
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

	if isLocal {
		// because we're local, go ahead and create the bucket
		err := devMakeBucket(ds.S3, bucket)
		if err != nil {
			panic(fmt.Errorf("error creating bucket: %v", err))
		}
	}

	return ds, nil
}

func devMakeBucket(s3obj *s3.S3, bucketName string) error {
	_, err := s3obj.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})

	return err
}
