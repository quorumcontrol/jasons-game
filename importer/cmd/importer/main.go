package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/importer"
	"github.com/quorumcontrol/jasons-game/network"
)

func main() {
	importPath := flag.String("path", "", "which directory to import")
	local := flag.Bool("local", true, "connect to localnet & use localstack S3 instead of testnet & real S3")
	logLevel := flag.String("log", "debug", "log level for importer")
	flag.Parse()

	err := logging.SetLogLevel("importer", *logLevel)
	if err != nil {
		panic(err)
	}

	if *importPath == "" {
		panic(fmt.Errorf("Must set -path to directory to import"))
	}

	var net network.Network
	// FIXME: this should be local / testnet, not mock / local
	if *local {
		net = network.NewLocalNetwork()
	} else {
		ctx := context.Background()

		group, err := network.SetupTupeloNotaryGroup(ctx, true)
		if err != nil {
			panic(errors.Wrap(err, "error setting up tupelo notary group"))
		}

		ds := dssync.MutexWrap(datastore.NewMapDatastore())

		net, err = network.NewRemoteNetwork(ctx, group, ds)
		if err != nil {
			panic(errors.Wrap(err, "error setting up remote network"))
		}
	}

	_, err = importer.New(net).Import(*importPath)
	if err != nil {
		panic(err)
	}
}
