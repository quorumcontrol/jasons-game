// +build desktop

package ui

import (
	"github.com/zserge/webview"
)

func OpenWebView() {
	webview.Open("Jason's Game",
		"http://localhost:8080", 1366, 768, true)
}
