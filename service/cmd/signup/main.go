package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/pkg/errors"
	"github.com/shibukawa/configdir"
	"github.com/spf13/cobra"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/game/signup"
)

func main() {
	var localNetworkFlag bool

	rootCmd := &cobra.Command{
		Use:   "jason-signup-service",
		Short: "jason-signup-service tracks signups for game",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var signingKey *ecdsa.PrivateKey
			var err error
			privateKeyHex, ok := os.LookupEnv("JASONS_GAME_ECDSA_KEY_HEX")
			if !ok {
				panic(errors.Wrap(err, "must set JASONS_GAME_ECDSA_KEY_HEX env var"))
			}

			signingKey, err = crypto.ToECDSA(hexutil.MustDecode(privateKeyHex))
			if err != nil {
				panic(errors.Wrap(err, "error decoding ecdsa key"))
			}

			notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, localNetworkFlag)
			if err != nil {
				panic(errors.Wrap(err, "error setting up tupelo notary group"))
			}

			networkKey, err := crypto.GenerateKey()
			if err != nil {
				panic(errors.Wrap(err, "error generate key"))
			}

			ds, err := badger.NewDatastore(storageDirFor("signups"), &badger.DefaultOptions)
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

			pid := actor.EmptyRootContext.Spawn(signup.NewServerProps(net))
			stopOnSignal(pid)
		},
	}

	rootCmd.Flags().BoolVar(&localNetworkFlag, "local", false, "should this use local tupelo/jason, defaults to false")
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
