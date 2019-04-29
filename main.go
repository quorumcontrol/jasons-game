package main

import (
	"fmt"

	"github.com/quorumcontrol/jasons-game/ui"
)

func main() {
	app, err := ui.New()
	if err != nil {
		panic(fmt.Errorf("error creating ui: %v", err))
	}

	if err := app.Run(); err != nil {
		panic(fmt.Errorf("error running UI: %v", err))
	}
}
