package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/server"
	"github.com/quorumcontrol/jasons-game/services"
	"github.com/quorumcontrol/jasons-game/handlers/inventory"
	"github.com/shibukawa/configdir"
	"github.com/spf13/cobra"
)

func main() {
	var localNetwork bool

	rootCmd := &cobra.Command{
		Use:   "jason-listener-service",
		Short: "jason-listener-service listens for events and performs actions",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			notaryGroup, err := server.SetupTupeloNotaryGroup(ctx, localNetwork)
			if err != nil {
				panic(errors.Wrap(err, "error setting up tupelo notary group"))
			}

			net, err := network.NewRemoteNetwork(ctx, notaryGroup, storageDirFor("defaultService"))
			if err != nil {
				panic(errors.Wrap(err, "setting up network"))
			}

			actorCtx := actor.EmptyRootContext
			servicePID := actorCtx.Spawn(services.NewServiceProps(net))

			inventoryAddProps := inventory.NewUnrestrictedAddHandler(net)
			actorCtx.Send(servicePID, &services.AttachHandler{HandlerProps: inventoryAddProps})

			inventoryRemoveProps := inventory.NewUnrestrictedRemoveHandler(net)
			actorCtx.Send(servicePID, &services.AttachHandler{HandlerProps: inventoryRemoveProps})

			stopOnSignal(servicePID)
		},
	}

	rootCmd.Flags().BoolVar(&localNetwork, "local", false, "should this use local tupelo/jason, defaults to false")
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
