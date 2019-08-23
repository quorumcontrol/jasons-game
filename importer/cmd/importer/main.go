package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/importer"
	"github.com/quorumcontrol/jasons-game/network"
)

func main() {
	err := logging.SetLogLevel("importer", "debug")
	if err != nil {
		panic(err)
	}

	importPath := flag.String("path", "", "which directory to import")
	local := flag.Bool("local", false, "connect to localnet & use localstack S3 instead of testnet & real S3")
	logLevel := flag.String("log", "debug", "log level for importer")
	flag.Parse()

	if *importPath == "" {
		panic(fmt.Errorf("Must set -path to directory to import"))
	}

	ctx := context.Background()

	err = logging.SetLogLevel("importer", *logLevel)
	if err != nil {
		panic(err)
	}

	var signingKey *ecdsa.PrivateKey
	privateKeyHex, ok := os.LookupEnv("JASONS_GAME_ECDSA_KEY_HEX")
	if ok {
		signingKey, err = crypto.ToECDSA(hexutil.MustDecode(privateKeyHex))
		if err != nil {
			panic(errors.Wrap(err, "error decoding ecdsa key"))
		}
	} else {
		signingKey, err = crypto.GenerateKey()
		if err != nil {
			panic(errors.Wrap(err, "error generate key"))
		}
	}

	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, *local)
	if err != nil {
		panic(errors.Wrap(err, "error setting up tupelo notary group"))
	}

	networkKey, err := crypto.GenerateKey()
	if err != nil {
		panic(errors.Wrap(err, "error generate key"))
	}

	// Just use a memory store and expect that nodes are stored
	// in the community blockstore from being signed
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	if err != nil {
		panic(errors.Wrap(err, "error creating store"))
	}

	config := &network.RemoteNetworkConfig{
		NotaryGroup:   notaryGroup,
		KeyValueStore: ds,
		SigningKey:    signingKey,
		NetworkKey:    networkKey,
	}

	net, err := network.NewRemoteNetworkWithConfig(ctx, config)
	if err != nil {
		panic(errors.Wrap(err, "setting up network"))
	}

	_, err = importer.New(net).Import(*importPath)
	if err != nil {
		panic(err)
	}
}
