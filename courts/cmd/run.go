package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/quorumcontrol/jasons-game/courts/autumn"
	"github.com/quorumcontrol/jasons-game/courts/spring"
	"github.com/quorumcontrol/jasons-game/courts/summer"
	"github.com/spf13/cobra"
)

var courtsList []string

type courtStarter interface {
	Start()
}

var runCourts = &cobra.Command{
	Use:   "run",
	Short: "Run one or more court handlers",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		net := setupNetwork(ctx, newFileStore("courts"), localNetworkFlag)

		if len(courtsList) == 0 {
			panic("must specify at least one --court")
		}

		for _, courtName := range courtsList {
			var court courtStarter

			switch courtName {
			case "autumn":
				court = autumn.New(ctx, net, configDir)
			case "spring":
				court = spring.New(ctx, net, configDir)
			case "summer":
				court = summer.New(ctx, net, configDir)
			default:
				panic("unknown court named " + courtName)
			}

			court.Start()
		}

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)

		select {
		case <-ctx.Done():
			fmt.Println("program exited within context")
		case sig := <-sigCh:
			fmt.Printf("program received signal %v\n", sig)
			cancel()
		}
	},
}

func init() {
	runCourts.Flags().StringSliceVar(&courtsList, "court", []string{}, "name(s) of court(s) to run")
	err := runCourts.MarkFlagRequired("court")
	if err != nil {
		panic(err)
	}
	rootCmd.AddCommand(runCourts)
}