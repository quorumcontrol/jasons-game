package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/shibukawa/configdir"

	bootstrap "github.com/quorumcontrol/jasons-game/jason/bootstrap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mustSetLogLevel("*", "warning")
	mustSetLogLevel("pubsub", "error")
	mustSetLogLevel("jasonbootstrap", "debug")

	ip := flag.String("ip", "0.0.0.0", "The IP address to bind jason to")
	port := flag.Int("port", 0, "Port to listen on, 0 means random port")
	flag.Parse()

	p2pOpts := []p2p.Option{
		p2p.WithListenIP(*ip, *port),
	}

	node, err := bootstrap.New(ctx, mustGetOrCreatePrivateKey(), p2pOpts...)
	if err != nil {
		panic(err)
	}
	err = node.Start()
	if err != nil {
		panic(err)
	}
	<-make(chan struct{})
}

func mustGetOrCreatePrivateKey() *ecdsa.PrivateKey {
	configDirs := configdir.New("tupelo", "jasons-game")
	folders := configDirs.QueryFolders(configdir.Global)

	folder := configDirs.QueryFolderContainsFile("private.key")
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
	return key
}

func mustSetLogLevel(name, level string) {
	if err := logging.SetLogLevel(name, level); err != nil {
		panic(errors.Wrapf(err, "error setting log level of %s to %s", name, level))
	}
}
