// +build internal
package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"

	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/messages/build/go/transactions"
)

const inkLocalName = "ink"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	local := flag.Bool("local", true, "connect to locally running notary group.")
	flag.Parse()

	var remoteOrLocal string
	if *local {
		remoteOrLocal = "local"
	} else {
		remoteOrLocal = "remote"
	}

	fmt.Printf("Setting up a %s notary group...\n", remoteOrLocal)
	group, err := network.SetupTupeloNotaryGroup(ctx, *local)
	if err != nil {
		panic(errors.Wrap(err, "error setting up notary group"))
	}

	fmt.Println("Generating a new ink chaintree private key...")
	signingKey, err := crypto.GenerateKey()
	if err != nil {
		panic(errors.Wrap(err, "error generating private key for ink faucet"))
	}

	fmt.Println("Setting up remote network...")
	ds := config.MemoryDataStore()
	netCfg := &network.RemoteNetworkConfig{
		NotaryGroup:   group,
		KeyValueStore: ds,
		SigningKey:    signingKey,
	}

	net, err := network.NewRemoteNetworkWithConfig(ctx, netCfg)
	if err != nil {
		panic(errors.Wrap(err, "error setting up remote network"))
	}

	fmt.Println("Creating new ink chaintree...")
	inkTree, err := net.CreateChainTreeWithKey(signingKey)
	if err != nil {
		panic(errors.Wrap(err, "error creating ink chaintree"))
	}

	fmt.Println("Establishing ink token...")
	establishInkTxn, err := chaintree.NewEstablishTokenTransaction(inkLocalName, 0)
	if err != nil {
		panic(errors.Wrap(err, "error building establish ink token transaction"))
	}

	txResp, err := net.Tupelo.PlayTransactions(inkTree, signingKey, []*transactions.Transaction{establishInkTxn})
	if err != nil {
		panic(errors.Wrap(err, "error playing establish ink token transaction on ink chaintree"))
	}

	err = net.TreeStore().SaveTreeMetadata(inkTree)
	if err != nil {
		panic(errors.Wrap(err, "error saving ink chaintree metadata"))
	}

	inkTreeDID, err := inkTree.Id()
	if err != nil {
		panic(errors.Wrap(err, "error fetching ink chaintree did"))
	}

	inkTokenName := consensus.TokenName{ChainTreeDID: inkTreeDID, LocalName: inkLocalName}

	fmt.Println("Successfully created ink chaintree. Save and secure the following output:")
	fmt.Println("=========================================================================")
	fmt.Printf("INK_TREE_KEY=%s\n", base64.StdEncoding.EncodeToString(crypto.FromECDSA(signingKey)))
	fmt.Printf("INK_TREE_DID=%s\n", inkTreeDID)
	fmt.Printf("INK_TREE_TIP=%s\n", txResp.Tip)
	fmt.Printf("INK_TOKEN=%s\n", inkTokenName.String())
	fmt.Println("=========================================================================")
}
