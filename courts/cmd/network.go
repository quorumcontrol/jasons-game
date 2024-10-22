package cmd

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ipfs/go-datastore"
	"github.com/pkg/errors"
	"github.com/shibukawa/configdir"

	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func setupNetwork(ctx context.Context, ds datastore.Batching, localNetwork bool) network.Network {
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

	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, localNetwork)
	if err != nil {
		panic(errors.Wrap(err, "error setting up tupelo notary group"))
	}

	networkKey, err := crypto.GenerateKey()
	if err != nil {
		panic(errors.Wrap(err, "error generate key"))
	}

	networkConfig := &network.RemoteNetworkConfig{
		NotaryGroup:   notaryGroup,
		KeyValueStore: ds,
		SigningKey:    signingKey,
		NetworkKey:    networkKey,
	}

	externalIP := os.Getenv("JASONS_GAME_EXTERNAL_IP")
	externalPortStr := os.Getenv("JASONS_GAME_EXTERNAL_PORT")

	if externalIP != "" && externalPortStr != "" {
		networkConfig.ExternalIP = externalIP
		networkConfig.ExternalPort, err = strconv.Atoi(externalPortStr)

		if err != nil {
			panic(errors.Wrap(err, "error parsing port, must be integer"))
		}

		// Generate static IPLD net key from private key + ip + port
		ipldKeySeed := sha256.Sum256([]byte(networkConfig.ExternalIP + externalPortStr))
		networkConfig.IpldKey, err = consensus.PassPhraseKey(crypto.FromECDSA(signingKey), ipldKeySeed[:32])
		if err != nil {
			panic(errors.Wrap(err, "error generate IPLD network key"))
		}
	}

	net, err := network.NewRemoteNetworkWithConfig(ctx, networkConfig)
	if err != nil {
		panic(errors.Wrap(err, "setting up network"))
	}

	return net
}

func newFileStore(name string) datastore.Batching {
	relativeConfigDir := filepath.Join("services", name)
	configDirs := configdir.New("tupelo", "jasons-game")
	folders := configDirs.QueryFolders(configdir.Global)
	folder := configDirs.QueryFolderContainsFile(relativeConfigDir)

	if folder == nil {
		if err := folders[0].CreateParentDir(filepath.Join(relativeConfigDir, "init")); err != nil {
			panic(err)
		}
	}

	ds, err := config.LocalDataStore(filepath.Join(folders[0].Path, relativeConfigDir))
	if err != nil {
		panic(errors.Wrap(err, "error creating store"))
	}

	return ds
}
