package config

import "github.com/shibukawa/configdir"

func Dir() *configdir.ConfigDir {
	configDir := configdir.New("tupelo", "jasons-game")
	return &configDir
}

func EnsureExists(name string) *configdir.Config {
	configDir := Dir()
	folders := configDir.QueryFolders(configdir.Global)
	folder := configDir.QueryFolderContainsFile(name)
	if folder == nil {
		if err := folders[0].CreateParentDir(name); err != nil {
			panic(err)
		}
	}

	return folder
}
