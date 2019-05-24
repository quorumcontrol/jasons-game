package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	logging "github.com/ipfs/go-log"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/jason/provider"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/shibukawa/configdir"
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

	ds, err := badger.NewDatastore(filepath.Join(folders[0].Path, jasonPath), &badger.DefaultOptions)
	if err != nil {
		panic(errors.Wrap(err, "error creating store"))
	}

	ip := flag.String("ip", "0.0.0.0", "The IP address to bind jason to")
	port := flag.Int("port", 0, "Port to listen on, 0 means random port")

	flag.Parse()
	fmt.Printf("ip %s port %d\n", *ip, *port)

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
