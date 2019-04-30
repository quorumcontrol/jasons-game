package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/ui"
)

func main() {
	rootCtx := actor.EmptyRootContext
	ui, err := rootCtx.SpawnNamed(ui.NewUIProps(), "ui")
	if err != nil {
		panic(fmt.Errorf("error running UI: %v", err))
	}
	gameActor, err := rootCtx.SpawnNamed(game.NewGameProps(ui, game.DefaultTree), "game")
	if err != nil {
		panic(fmt.Errorf("error running UI: %v", err))
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()
	fmt.Println("hit ctrl-C one more time to exit")
	<-done
	gameActor.Stop()
}
