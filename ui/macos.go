// +build macos_app_bundle

package ui

func SetOptions() (disableWebView *bool, localnet *bool) {
	imFalse := false

	disableWebView = &imFalse
	localnet = &imFalse

	return
}
