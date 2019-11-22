package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
)

var localNetworkFlag bool

var courtDids = map[string]string{
	"arcadia": "did:tupelo:0x8daEDC229EfC5Cd508A70aff74bbfaC0D5762af4",
	"autumn":  "did:tupelo:0x4Fad913ee0f39D761B25251Ad3986714830CD265",
	"spring":  "did:tupelo:0x18540E9b8691136F881Dfc6D231c3D2CEB3667d2",
	"summer":  "did:tupelo:0xf74F1beC81E6Dcbc9540dD54FeC27349925eF291",
	"winter":  "did:tupelo:0x5d407B1E1623197E40c284Dbc162bbc2e827eA7A",
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	net := setupNetwork(ctx)

	for name, did := range courtDids {
		tree, err := net.GetTree(did)
		if err != nil {
			panic(err)
		}

		f, err := os.Create(fmt.Sprintf("winners_%s.csv", name))
		if err != nil {
			panic(err)
		}
		defer f.Close()

		_, err = f.Write([]byte("num,player DID,prize DID,signup key\n"))
		if err != nil {
			panic(err)
		}

		winnersPath := []string{"tree", "data", "jasons-game", "winners"}

		bucketsUncast, _, err := tree.ChainTree.Dag.Resolve(ctx, winnersPath)
		if err != nil {
			panic(err)
		}

		buckets := sortNumericKeys(bucketsUncast)

		for _, bucket := range buckets {
			winnersListUncast, _, err := tree.ChainTree.Dag.Resolve(ctx, append(winnersPath, bucket))
			if err != nil {
				panic(err)
			}

			winnersList := sortNumericKeys(winnersListUncast)

			for _, winnerNum := range winnersList {
				winner, _, err := tree.ChainTree.Dag.Resolve(ctx, append(winnersPath, bucket, winnerNum, "player", "id"))
				if err != nil {
					panic(err)
				}

				prize, _, err := tree.ChainTree.Dag.Resolve(ctx, append(winnersPath, bucket, winnerNum, "prize", "id"))
				if err != nil {
					panic(err)
				}

				winnerTree, err := net.GetTree(winner.(string))
				if err != nil {
					panic(err)
				}

				auths, err := winnerTree.Authentications()
				if err != nil {
					panic(err)
				}

				_, err = f.Write([]byte(strings.Join([]string{winnerNum, winner.(string), prize.(string), auths[0]}, ",") + "\n"))
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

func sortNumericKeys(val interface{}) []string {
	if val == nil {
		return []string{}
	}

	valMap := val.(map[string]interface{})
	sorted := make([]int, len(valMap))

	i := 0
	for num := range val.(map[string]interface{}) {
		int, err := strconv.Atoi(num)
		if err != nil {
			panic(err)
		}
		sorted[i] = int
		i++
	}
	sort.Ints(sorted)

	asStr := make([]string, len(sorted))
	for i, num := range sorted {
		asStr[i] = strconv.Itoa(num)
	}
	return asStr
}

func setupNetwork(ctx context.Context) *network.RemoteNetwork {
	var err error

	signingKey, err := crypto.GenerateKey()
	if err != nil {
		panic(errors.Wrap(err, "error generate key"))
	}

	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, localNetworkFlag)
	if err != nil {
		panic(errors.Wrap(err, "error setting up tupelo notary group"))
	}

	networkKey, err := crypto.GenerateKey()
	if err != nil {
		panic(errors.Wrap(err, "error generate key"))
	}

	config := &network.RemoteNetworkConfig{
		NotaryGroup:   notaryGroup,
		KeyValueStore: config.MemoryDataStore(),
		SigningKey:    signingKey,
		NetworkKey:    networkKey,
	}

	net, err := network.NewRemoteNetworkWithConfig(ctx, config)
	if err != nil {
		panic(errors.Wrap(err, "setting up network"))
	}

	return net
}
