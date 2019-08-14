package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/quorumcontrol/jasons-game/courts/summer"
	"github.com/spf13/cobra"
)

var summerCourt = &cobra.Command{
	Use:   "summer",
	Short: "Run the summer court handler",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		net := setupNetwork(ctx, newFileStore("summer"), localNetworkFlag)

		s := summer.New(ctx, net, configDir)
		if err != nil {
			panic(err)
		}
		s.Start()

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
	rootCmd.AddCommand(summerCourt)
}
