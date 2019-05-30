// +build !desktop

package ui

func OpenWebView() {
	panic("unsupported, you must compile the binary using -tags=desktop")
}
