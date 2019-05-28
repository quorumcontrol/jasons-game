// +build !macos_app_bundle

package ui

import "flag"

func SetOptions() (disableWebView bool, localnet bool) {
	disableWebView = *flag.Bool("disablewebview", false, "disable the webview")
	localnet = *flag.Bool("localnet", false, "connect to localnet instead of testnet")

	flag.Parse()

	return
}
