package main

import (
	"context"
	"flag"
	"fmt"

	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/inkwell/server"
)

var log = logging.Logger("inkwell")

const stateStorageDir = "state-storage"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	localnet := flag.Bool("localnet", false, "connect to localnet instead of testnet")
	flag.Parse()

    stateCfg := config.EnsureExists(stateStorageDir)

	serverCfg := server.InkwellConfig{
		Localnet:  *localnet,
		StatePath: stateCfg.Path,
	}

	inkwell, err := server.NewServer(ctx, serverCfg)
	if err != nil {
		panic(errors.Wrap(err, "error creating new inkwell server"))
	}

	// TODO: Call some kind of inkwell.Start() to make it listen for player invites on a topic
	fmt.Printf("Started inkwell %+v\n", inkwell)

	<-make(chan struct{})
}
