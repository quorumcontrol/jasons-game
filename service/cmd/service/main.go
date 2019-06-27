package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
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

			notaryGroup, err := server.SetupTupeloNotaryGroup(ctx, localNetworkFlag)
			if err != nil {
				panic(errors.Wrap(err, "error setting up tupelo notary group"))
			}

			net, err := network.NewRemoteNetwork(ctx, notaryGroup, storageDirFor(serviceName))
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
			serviceDid, err := actor.EmptyRootContext.RequestFuture(servicePID, &service.GetServiceDid{}, 5 * time.Second).Result()
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
	rootCmd.MarkFlagRequired("handlers")
	rootCmd.Execute()
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
