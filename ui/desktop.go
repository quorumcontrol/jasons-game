// +build desktop

package ui

import (
	"github.com/zserge/webview"
)

func OpenWebView() {
	webview.Open("Jason's Game",
		"http://localhost:8080", 800, 600, true)
}
