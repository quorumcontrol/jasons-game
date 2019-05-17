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
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/jason/provider"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/shibukawa/configdir"
)

var log = logging.Logger("jason")

const minConnections = 4915 // 60% of 8192 ulimit
const maxConnections = 7372 // 90% of 8192 ulimit
const connectionGracePeriod = 20 * time.Second

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logging.SetLogLevel("*", "warning")
	logging.SetLogLevel("pubsub", "error")
	logging.SetLogLevel("jasonblocks", "debug")

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
		ioutil.WriteFile(fullPath, []byte(hexutil.Encode(crypto.FromECDSA(key))), 0600)
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

	ds, err := badger.NewDatastore(filepath.Join(folders[0].Path, "storage"), &badger.DefaultOptions)
	if err != nil {
		panic(errors.Wrap(err, "error creating store"))
	}

	ip := flag.String("ip", "0.0.0.0", "The IP address to bind jason to")
	port := flag.Int("port", 0, "Port to listen on, 0 means random port")

	flag.Parse()
	fmt.Printf("ip %s port %d\n", *ip, *port)

	cm := connmgr.NewConnManager(minConnections, maxConnections, connectionGracePeriod)

	p2pOpts := []p2p.Option{
		p2p.WithListenIP(*ip, *port),
		p2p.WithLibp2pOptions(libp2p.ConnectionManager(cm)),
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
