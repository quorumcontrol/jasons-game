package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/pkg/errors"
	"github.com/shibukawa/configdir"
	"github.com/spf13/cobra"

	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/service"
	endgame "github.com/quorumcontrol/jasons-game/service/cmd/endgame/handlers"

	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func main() {
	var localNetworkFlag bool
	var serviceName string

	rootCmd := &cobra.Command{
		Use:   "endgame-listener-service",
		Short: "endgame-listener-service listens on endgame altars",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var signingKey *ecdsa.PrivateKey
			var err error
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

			notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, localNetworkFlag)
			if err != nil {
				panic(errors.Wrap(err, "error setting up tupelo notary group"))
			}

			networkKey, err := crypto.GenerateKey()
			if err != nil {
				panic(errors.Wrap(err, "error generate key"))
			}

			ds, err := badger.NewDatastore(storageDirFor(serviceName), &badger.DefaultOptions)
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

			summonTree, err := net.CreateChainTree()
			if err != nil {
				panic(errors.Wrap(err, "creating summon tree"))
			}

			fmt.Printf("summoning room = %s\n", summonTree.MustId())

			altarColors := []string{
				"blue",
				"red",
				"yellow",
			}
			altarMaterials := []string{
				"steel",
				"wood",
				"stone",
			}

			altars := make([]*consensus.SignedChainTree, len(altarColors))
			for i := 0; i < len(altars); i++ {
				tree, err := net.CreateChainTree()
				fmt.Printf("altar %d = %s\n", i, tree.MustId())
				if err != nil {
					panic(errors.Wrap(err, "error creating tree"))
				}

				altars[i] = tree
			}

			for i, altarTree := range altars {
				loc := game.NewLocationTree(net, altarTree)
				err = loc.SetDescription(fmt.Sprintf("This is altar %d", i+1))
				if err != nil {
					panic(errors.Wrap(err, "error creating tree"))
				}

				err = loc.AddInteraction(&game.ChangeLocationInteraction{
					Command: "enter summoning room",
					Did:     summonTree.MustId(),
				})
				if err != nil {
					panic(errors.Wrap(err, "error creating tree"))
				}

				if i > 0 {
					err = loc.AddInteraction(&game.ChangeLocationInteraction{
						Command: "previous",
						Did:     altars[i-1].MustId(),
					})
					if err != nil {
						panic(errors.Wrap(err, "error creating tree"))
					}
				}

				if i < (len(altars) - 1) {
					err = loc.AddInteraction(&game.ChangeLocationInteraction{
						Command: "next",
						Did:     altars[i+1].MustId(),
					})
					if err != nil {
						panic(errors.Wrap(err, "error creating tree"))
					}
				}
			}

			altarConfigs := make([]*endgame.EndGameAltar, len(altars))
			for i, altarTree := range altars {

				altarConfigs[i] = &endgame.EndGameAltar{
					Did: altarTree.MustId(),
					Requires: []string{
						fmt.Sprintf("color %s", altarColors[i]),
						fmt.Sprintf("material %s", altarMaterials[i]),
					},
				}
			}

			altarHandler := endgame.NewEndGameAltarHandler(net, altarConfigs)
			summonHandler := endgame.NewEndGameSummonHandler(net, summonTree.MustId(), altarConfigs)

			handler := handlers.NewCompositeHandler([]handlers.Handler{altarHandler, summonHandler})

			servicePID := actor.EmptyRootContext.Spawn(service.NewServiceActorProps(net, handler))
			serviceDid, err := actor.EmptyRootContext.RequestFuture(servicePID, &service.GetServiceDid{}, 30*time.Second).Result()
			if err != nil {
				panic(err)
			}
			fmt.Printf("Starting service with ChainTree id %v\n", serviceDid)

			for _, altarTree := range altars {
				loc := game.NewLocationTree(net, altarTree)

				err = loc.SetHandler(serviceDid.(string))
				if err != nil {
					panic(errors.Wrap(err, "error creating tree"))
				}
			}

			summonLoc := game.NewLocationTree(net, summonTree)
			err = summonLoc.SetHandler(serviceDid.(string))
			if err != nil {
				panic(errors.Wrap(err, "error creating tree"))
			}

			for i, altarTree := range altars {
				err = summonLoc.AddInteraction(&game.ChangeLocationInteraction{
					Command: fmt.Sprintf("return to altar %d", i),
					Did:     altarTree.MustId(),
				})
				if err != nil {
					panic(errors.Wrap(err, "error creating tree"))
				}
			}

			stopOnSignal(servicePID)
		},
	}

	rootCmd.Flags().BoolVar(&localNetworkFlag, "local", false, "should this use local tupelo/jason, defaults to false")
	rootCmd.Flags().StringVar(&serviceName, "name", "defaultService", "unique name of this service")
	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}

func storageDirFor(name string) string {
	relativeConfigDir := filepath.Join("services", name)
	configDirs := configdir.New("tupelo", "jasons-game")
	folders := configDirs.QueryFolders(configdir.Global)
	folder := configDirs.QueryFolderContainsFile(relativeConfigDir)
	if folder == nil {
		if err := folders[0].CreateParentDir(filepath.Join(relativeConfigDir, "init")); err != nil {
			panic(err)
		}
	}
	return filepath.Join(folders[0].Path, relativeConfigDir)
}

func stopOnSignal(actors ...*actor.PID) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		for _, act := range actors {
			err := actor.EmptyRootContext.PoisonFuture(act).Wait()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Actor failed to stop gracefully: %s\n", err)
			}
		}
		done <- true
	}()
	<-done
}
