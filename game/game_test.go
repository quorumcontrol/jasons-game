package game

// func TestNavigation(t *testing.T) {
// 	rootCtx := actor.EmptyRootContext
// 	simulatedUI, err := rootCtx.SpawnNamed(ui.NewSimulatedUIProps(), "test-navigation-ui")
// 	require.Nil(t, err)
// 	defer simulatedUI.Stop()

// 	net := network.NewLocalNetwork()

// 	game, err := rootCtx.SpawnNamed(NewGameProps(simulatedUI, net), "test-navigation-game")
// 	require.Nil(t, err)
// 	defer game.Stop()

// 	rootCtx.Send(game, &ui.UserInput{Message: "north"})
// 	time.Sleep(100 * time.Millisecond)
// 	fut := rootCtx.RequestFuture(simulatedUI, &ui.GetEventsFromSimulator{}, 1*time.Second)
// 	evts, err := fut.Result()

// 	assert.Len(t, evts.([]interface{}), 2)
// 	assert.IsType(t, &navigator.Location{}, evts.([]interface{})[0])
// 	assert.IsType(t, &navigator.Location{}, evts.([]interface{})[1])
// }
