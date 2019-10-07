package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var localNetworkFlag bool
var logLevel string
var configDir string

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "courts",
	Short: "courts service for game interactions in each court",
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&localNetworkFlag, "local", false, "should this use local tupelo/jason, defaults to false")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "L", "info", "logging level (debug|info|warn|error)")
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "", "path to yaml config directory")
	if err := rootCmd.MarkPersistentFlagRequired("config"); err != nil {
		panic(err)
	}
}
