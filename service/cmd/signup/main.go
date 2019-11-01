package main

import (
	"bytes"
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
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/gogo/protobuf/proto"
	"github.com/ipfs/go-cid"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/pkg/errors"
	"github.com/shibukawa/configdir"
	"github.com/spf13/cobra"

	"github.com/quorumcontrol/jasons-game/game/signup"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

var localNetworkFlag bool

func setupNetwork(ctx context.Context) *network.RemoteNetwork {
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

	return net
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "jason-signup-service",
		Short: "jason-signup-service tracks signups for game",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	rootCmd.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "Run the signup listener service",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			net := setupNetwork(ctx)
			pid := actor.EmptyRootContext.Spawn(signup.NewServerProps(net))
			stopOnSignal(pid)
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "export",
		Short: "export all emails & dids to signup.csv",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			net := setupNetwork(ctx)

			signupClient, err := signup.NewClient(net)
			if err != nil {
				panic(err)
			}

			encryptionPubKey, err := signupClient.EncryptionPubKey()
			if err != nil {
				panic(err)
			}

			encryptionPubKeyBytes := crypto.FromECDSAPub(encryptionPubKey)

			if !bytes.Equal(encryptionPubKeyBytes, crypto.FromECDSAPub(net.PublicKey())) {
				panic(fmt.Sprintf("wrong JASONS_GAME_ECDSA_KEY_HEX, use the private key matching the public key %s", hexutil.Encode(encryptionPubKeyBytes)))
			}

			tree, err := net.GetTree(signupClient.Did())
			if err != nil {
				panic(err)
			}

			trackingUncast, _, err := tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "track"})
			if err != nil {
				panic(err)
			}

			f, err := os.Create("signups.csv")
			if err != nil {
				panic(err)
			}
			defer f.Close()

			_, err = f.Write([]byte("email,playerDid\n"))
			if err != nil {
				panic(err)
			}

			recursiveSignupExport(ctx, net, f, trackingUncast)
		},
	})

	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}

func recursiveSignupExport(ctx context.Context, net *network.RemoteNetwork, f *os.File, data interface{}) {
	if data == nil {
		panic("empty data")
	}

	dataAsMap, ok := data.(map[string]interface{})
	if !ok {
		panic("bad map data")
	}

	for _, v := range dataAsMap {
		switch asType := v.(type) {
		case cid.Cid:
			someNode, err := net.Ipld().Get(ctx, asType)
			if err != nil {
				panic(err)
			}
			resolved, _, err := someNode.Resolve([]string{})
			if err != nil {
				panic(err)
			}
			recursiveSignupExport(ctx, net, f, resolved)
		case []byte:
			eciesKey := ecies.ImportECDSA(net.PrivateKey())
			decrypted, err := eciesKey.Decrypt(asType, nil, nil)
			if err != nil {
				panic(errors.Wrap(err, "decryption failed, probably caused by incorrect JASONS_GAME_ECDSA_KEY_HEX"))
			}

			signup := &jasonsgame.SignupMessage{}
			err = proto.Unmarshal(decrypted, signup)
			if err != nil {
				panic(err)
			}

			_, err = f.Write([]byte(signup.GetEmail() + "," + signup.GetDid() + "\n"))
			if err != nil {
				panic(err)
			}
		default:
			fmt.Printf("Unknown value %v", v)
		}
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
