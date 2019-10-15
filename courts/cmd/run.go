package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/courts/arcadia"
	"github.com/quorumcontrol/jasons-game/courts/autumn"
	"github.com/quorumcontrol/jasons-game/courts/basic"
	"github.com/quorumcontrol/jasons-game/courts/spring"
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

		config.MustSetLogLevel("importer", logLevel)
		config.MustSetLogLevel("respawner", logLevel)
		config.MustSetLogLevel("swarm2", logLevel)
		config.MustSetLogLevel("pubsub", logLevel)
		config.MustSetLogLevel("autonat", logLevel)
		config.MustSetLogLevel("dht", logLevel)
		config.MustSetLogLevel("bitswap", logLevel)
		config.MustSetLogLevel("gamenetwork", logLevel)

		net := setupNetwork(ctx, newFileStore("courts"), localNetworkFlag)

		if len(courtsList) == 0 {
			panic("must specify at least one --court")
		}

		go func() {
			debugR := mux.NewRouter()
			debugR.HandleFunc("/debug/pprof/", pprof.Index)
			debugR.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			debugR.HandleFunc("/debug/pprof/profile", pprof.Profile)
			debugR.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			debugR.Handle("/debug/pprof/heap", pprof.Handler("heap"))
			debugR.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
			debugR.Handle("/debug/pprof/block", pprof.Handler("block"))
			debugR.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
			err := http.ListenAndServe(":8080", debugR)
			if err != nil {
				fmt.Println(err.Error())
			}
		}()

		fmt.Printf("Court authentication address is %s\n", crypto.PubkeyToAddress(*net.PublicKey()).String())

		for _, courtName := range courtsList {
			var court courtStarter

			switch courtName {
			case "arcadia":
				court = arcadia.New(ctx, net, configDir)
			case "autumn":
				court = autumn.New(ctx, net, configDir)
			case "spring":
				court = spring.New(ctx, net, configDir)
			case "summer":
				court = basic.New(ctx, net, configDir, "summer")
			case "winter":
				court = basic.New(ctx, net, configDir, "winter")
			default:
				panic("unknown court named " + courtName)
			}
			config.MustSetLogLevel(courtName, logLevel)
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
