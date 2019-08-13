// +build !macos_app_bundle

package ui

import "flag"

func SetOptions() (localnet *bool) {
	localnet = flag.Bool("localnet", false, "connect to localnet instead of testnet")

	flag.Parse()

	return
}
