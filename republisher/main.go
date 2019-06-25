package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/server"

	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

	"github.com/shibukawa/configdir"
)

const sessionStorageDir = "session-storage"

func doIt(ctx context.Context) error {
	err := logging.SetLogLevel("gamenetwork", "info")
	if err != nil {
		return errors.Wrap(err, "error setting log level")
	}

	group, err := server.SetupTupeloNotaryGroup(ctx, false)
	if err != nil {
		panic(errors.Wrap(err, "setting up notary group"))
	}

	configDirs := configdir.New("tupelo", "jasons-game")
	folders := configDirs.QueryFolders(configdir.Global)
	folder := configDirs.QueryFolderContainsFile(sessionStorageDir)
	if folder == nil {
		if err := folders[0].CreateParentDir(sessionStorageDir); err != nil {
			panic(err)
		}
	}

	sessionPath := filepath.Join(folders[0].Path, sessionStorageDir)

	statePath := filepath.Join(sessionPath, filepath.Base("12345"))
	if err := os.MkdirAll(statePath, 0750); err != nil {
		panic(errors.Wrap(err, "error creating session storage"))
	}
	net, err := network.NewRemoteNetwork(ctx, group, statePath)
	if err != nil {
		panic(errors.Wrap(err, "setting up network"))
	}

	return net.(*network.RemoteNetwork).RepublishAll()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := doIt(ctx)
	if err != nil {
		panic(err)
	}
}
