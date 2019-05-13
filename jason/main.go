package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"

	logging "github.com/ipfs/go-log"

	"github.com/ethereum/go-ethereum/crypto"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/jason/provider"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/shibukawa/configdir"
)

var log = logging.Logger("jason")

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logging.SetLogLevel("*", "info")

	configDirs := configdir.New("tupelo", "jasons-game")
	folders := configDirs.QueryFolders(configdir.Global)
	folder := configDirs.QueryFolderContainsFile("storage")
	if folder == nil {
		folders[0].CreateParentDir("storage")
	}

	folder = configDirs.QueryFolderContainsFile("private.key")
	if folder == nil {
		folders[0].Create("private.key")
		fullPath := filepath.Join(folders[0].Path, "private.key")
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(errors.Wrap(err, "error generating key"))
		}
		ioutil.WriteFile(fullPath, crypto.FromECDSA(key), 0600)
	}
	folder = configDirs.QueryFolderContainsFile("private.key")
	bits, err := folder.ReadFile("private.key")
	if err != nil {
		panic(errors.Wrap(err, "error reading key"))
	}
	key, err := crypto.ToECDSA(bits)
	if err != nil {
		panic(errors.Wrap(err, "error unmarshaling key"))
	}

	ds, err := badger.NewDatastore(filepath.Join(folders[0].Path, "storage"), &badger.DefaultOptions)
	if err != nil {
		panic(errors.Wrap(err, "error creating store"))
	}

	ip := flag.String("ip", "0.0.0.0", "The IP address to bind jason to")
	port := flag.Int("port", 0, "Port to listen on, 0 means random port")

	flag.Parse()

	fmt.Printf("ip %s port %d\n", *ip, *port)

	p, err := provider.New(ctx, key, ds, p2p.WithListenIP(*ip, *port))
	if err != nil {
		panic(errors.Wrap(err, "error creating provider"))
	}
	err = p.Start()
	if err != nil {
		panic(errors.Wrap(err, "error startingprovider"))
	}

	<-make(chan struct{})
}
