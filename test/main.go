package main

import (
	"context"
	"fmt"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("testmain")

func main() {
	logging.SetLogLevel("*", "INFO")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cli, err := network.NewIPLDClient(ctx, datastore.NewMapDatastore())
	panicErr(err)
	sw := &safewrap.SafeWrap{}
	n := sw.WrapObject(map[string]string{"hi": "world", "im": "tupelohere"})
	panicErr(sw.Err)
	fmt.Printf("putting %s\n", n.Cid().String())
	cli.Add(ctx, n)
	<-make(chan struct{})
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
