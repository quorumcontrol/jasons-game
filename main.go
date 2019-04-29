package main

import (
	"fmt"

	"github.com/quorumcontrol/jasons-game/game"
)

func main() {
	g := game.New(game.DefaultTree)

	if err := g.UI.Run(); err != nil {
		panic(fmt.Errorf("error running UI: %v", err))
	}
}
