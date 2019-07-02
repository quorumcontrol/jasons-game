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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common/hexutil"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/handlers/inventory"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/server"
	"github.com/quorumcontrol/jasons-game/service"
	"github.com/shibukawa/configdir"
	"github.com/spf13/cobra"
)

func main() {
	var localNetworkFlag bool
	var handlersFlag []string
	var serviceName string

	rootCmd := &cobra.Command{
		Use:   "jason-listener-service",
		Short: "jason-listener-service listens for events and performs actions",
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

			notaryGroup, err := server.SetupTupeloNotaryGroup(ctx, localNetworkFlag)
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
				NotaryGroup: notaryGroup,
				KeyValueStore: ds,
				SigningKey: signingKey,
				NetworkKey: networkKey,
			}

			net, err := network.NewRemoteNetworkWithConfig(ctx, config)
			if err != nil {
				panic(errors.Wrap(err, "setting up network"))
			}

			serviceHandlers := []handlers.Handler{}

			for _, h := range handlersFlag {
				switch h {
				case "inventory.UnrestrictedAddHandler":
					serviceHandlers = append(serviceHandlers, inventory.NewUnrestrictedAddHandler(net))
				case "inventory.UnrestrictedRemoveHandler":
					serviceHandlers = append(serviceHandlers, inventory.NewUnrestrictedRemoveHandler(net))
				default:
					panic(fmt.Sprintf("handler of type %v is not supported", h))
				}
			}

			if len(serviceHandlers) == 0 {
				panic("must set at least one handler")
			}

			servicePID := actor.EmptyRootContext.Spawn(service.NewServiceActorProps(net, handlers.NewCompositeHandler(serviceHandlers)))
			serviceDid, err := actor.EmptyRootContext.RequestFuture(servicePID, &service.GetServiceDid{}, 30*time.Second).Result()
			if err != nil {
				panic(err)
			}
			fmt.Printf("Starting service with ChainTree id %v\n", serviceDid)

			stopOnSignal(servicePID)
		},
	}

	rootCmd.Flags().BoolVar(&localNetworkFlag, "local", false, "should this use local tupelo/jason, defaults to false")
	rootCmd.Flags().StringVar(&serviceName, "name", "defaultService", "uniquee name of this service")
	rootCmd.Flags().StringArrayVar(&handlersFlag, "handlers", []string{}, "what handlers to use for this service")
	err := rootCmd.MarkFlagRequired("handlers")
	if err != nil {
		panic(err)
	}
	err = rootCmd.Execute()
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
