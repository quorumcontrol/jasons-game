package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	logging "github.com/ipfs/go-log"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	s3ds "github.com/ipfs/go-ds-s3"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/jason/provider"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/shibukawa/configdir"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func mustSetLogLevel(name, level string) {
	if err := logging.SetLogLevel(name, level); err != nil {
		panic(errors.Wrapf(err, "error setting log level of %s to %s", name, level))
	}
}

const jasonPath = "jason"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mustSetLogLevel("*", "warning")
	mustSetLogLevel("pubsub", "error")
	mustSetLogLevel("jasonblocks", "debug")

	ip := flag.String("ip", "0.0.0.0", "The IP address to bind jason to")
	port := flag.Int("port", 0, "Port to listen on, 0 means random port")
	isLocal := flag.Bool("local", false, "turn on localmode config for s3")
	flag.Parse()
	fmt.Printf("ip %s port %d\n", *ip, *port)

	configDirs := configdir.New("tupelo", "jasons-game")
	folders := configDirs.QueryFolders(configdir.Global)
	folder := configDirs.QueryFolderContainsFile(jasonPath)
	if folder == nil {
		if err := folders[0].CreateParentDir(jasonPath); err != nil {
			panic(err)
		}
	}

	folder = configDirs.QueryFolderContainsFile("private.key")
	if folder == nil {
		if _, err := folders[0].Create("private.key"); err != nil {
			panic(err)
		}
		fullPath := filepath.Join(folders[0].Path, "private.key")
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(errors.Wrap(err, "error generating key"))
		}
		if err := ioutil.WriteFile(fullPath, []byte(hexutil.Encode(crypto.FromECDSA(key))), 0600); err != nil {
			panic(err)
		}
	}
	folder = configDirs.QueryFolderContainsFile("private.key")
	keyHexBytes, err := folder.ReadFile("private.key")
	if err != nil {
		panic(errors.Wrap(err, "error reading key"))
	}
	key, err := crypto.ToECDSA(hexutil.MustDecode(strings.TrimSpace(string(keyHexBytes))))
	if err != nil {
		panic(errors.Wrap(err, "error unmarshaling key"))
	}

	var config s3ds.Config
	if *isLocal {
		// first sleep to give the localstack docker container time to startup
		time.Sleep(2 * time.Second)
		config = s3ds.Config{
			RegionEndpoint: "http://localstack:4572",
			Bucket:         "tupelo-jason-blocks-local",
			Region:         "us-east-1",
			AccessKey:      "localonlyac",
			SecretKey:      "localonlysk",
		}
		// because we're local, go ahead and create the bucket
		err := devMakeBucket(config)
		if err != nil {
			panic(fmt.Errorf("error creating bucket: %v", err))
		}
	} else {
		panic("only local is supported")
	}

	ds, err := s3ds.NewS3Datastore(config)

	// ds, err := badger.NewDatastore(filepath.Join(folders[0].Path, jasonPath), &badger.DefaultOptions)
	if err != nil {
		panic(errors.Wrap(err, "error creating store"))
	}

	p2pOpts := []p2p.Option{
		p2p.WithListenIP(*ip, *port),
	}

	p, err := provider.New(ctx, key, ds, p2pOpts...)
	if err != nil {
		panic(errors.Wrap(err, "error creating provider"))
	}
	err = p.Start()
	if err != nil {
		panic(errors.Wrap(err, "error starting provider"))
	}

	<-make(chan struct{})
}

func devMakeBucket(conf s3ds.Config) error {
	awsConfig := aws.NewConfig()
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create new session: %s", err)
	}

	creds := credentials.NewChainCredentials([]credentials.Provider{
		&credentials.StaticProvider{Value: credentials.Value{
			AccessKeyID:     conf.AccessKey,
			SecretAccessKey: conf.SecretKey,
			SessionToken:    conf.SessionToken,
		}},
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{},
		&ec2rolecreds.EC2RoleProvider{Client: ec2metadata.New(sess)},
	})

	if conf.RegionEndpoint != "" {
		awsConfig.WithS3ForcePathStyle(true)
		awsConfig.WithEndpoint(conf.RegionEndpoint)
	}

	awsConfig.WithCredentials(creds)
	awsConfig.WithRegion(conf.Region)

	sess, err = session.NewSession(awsConfig)
	if err != nil {
		return fmt.Errorf("failed to create new session with aws config: %s", err)
	}
	s3obj := s3.New(sess)

	_, err = s3obj.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(conf.Bucket),
	})

	return err
}
