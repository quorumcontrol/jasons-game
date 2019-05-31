package game

// func TestInventoryActor_CreateObject(t *testing.T) {
// 	// setup

// 	context := actor.EmptyRootContext

// 	net := network.NewLocalNetwork()

// 	playerChainTree, err := net.CreateNamedChainTree("player")
// 	require.Nil(t, err)

// 	testPlayer := NewPlayerTree(net, playerChainTree)

// 	createObject, err := context.SpawnNamed(NewInventoryActorProps(&InventoryActorConfig{
// 		Player:  testPlayer,
// 		Network: net,
// 	}), "testCreateObject")
// 	require.Nil(t, err)
// 	defer context.Stop(createObject)

// 	// create first object

// 	response, err := context.RequestFuture(createObject, &CreateObjectRequest{Name: "test", Description: "test object"}, 1*time.Second).Result()
// 	require.Nil(t, err)

// 	createObjectResponse, ok := response.(*CreateObjectResponse)
// 	require.True(t, ok)
// 	require.Nil(t, createObjectResponse.Error)

// 	objectsPath, _ := consensus.DecodePath(ObjectsPath)
// 	objectPath := append(objectsPath, "test")
// 	playerObjNode, remainingPath, err := testPlayer.ChainTree().ChainTree.Dag.Resolve(append([]string{"tree", "data"}, objectPath...))
// 	require.Nil(t, err)
// 	require.Empty(t, remainingPath)

// 	objectChainTree, err := net.GetChainTreeByName("object:test")
// 	require.Nil(t, err)

// 	obj := Object{Did: objectChainTree.MustId()}
// 	assert.Equal(t, obj.Did, playerObjNode.(string))
// 	assert.Equal(t, obj.Did, createObjectResponse.Object.Did)

// 	netObj := NetworkObject{Object: obj, Network: net}

// 	name, err := netObj.Name()
// 	require.Nil(t, err)
// 	assert.Equal(t, "test", name)

// 	desc, err := netObj.Description()
// 	require.Nil(t, err)
// 	assert.Equal(t, "test object", desc)

// 	// create second object

// 	response, err = context.RequestFuture(createObject, &CreateObjectRequest{Name: "sword", Description: "ultimate sword"}, 1*time.Second).Result()
// 	require.Nil(t, err)

// 	createObjectResponse, ok = response.(*CreateObjectResponse)
// 	require.True(t, ok)
// 	require.Nil(t, createObjectResponse.Error)

// 	objectPath = append(objectsPath, "sword")
// 	playerObjNode, remainingPath, err = testPlayer.ChainTree().ChainTree.Dag.Resolve(append([]string{"tree", "data"}, objectPath...))
// 	require.Nil(t, err)
// 	require.Empty(t, remainingPath)

// 	objectChainTree, err = net.GetChainTreeByName("object:sword")
// 	require.Nil(t, err)

// 	obj = Object{Did: objectChainTree.MustId()}
// 	assert.Equal(t, obj.Did, playerObjNode.(string))
// 	assert.Equal(t, obj.Did, createObjectResponse.Object.Did)
// 	netObj = NetworkObject{Object: obj, Network: net}

// 	name, err = netObj.Name()
// 	require.Nil(t, err)
// 	assert.Equal(t, "sword", name)

// 	desc, err = netObj.Description()
// 	require.Nil(t, err)
// 	assert.Equal(t, "ultimate sword", desc)
// }

// func TestInventoryActor_Receive_NamesMustBeUnique(t *testing.T) {
// 	// setup

// 	context := actor.EmptyRootContext

// 	net := network.NewLocalNetwork()

// 	playerChainTree, err := net.CreateNamedChainTree("player")
// 	require.Nil(t, err)

// 	testPlayer := NewPlayerTree(net, playerChainTree)

// 	createObject, err := context.SpawnNamed(NewInventoryActorProps(&InventoryActorConfig{
// 		Player:  testPlayer,
// 		Network: net,
// 	}), "testCreateObject")
// 	require.Nil(t, err)
// 	defer context.Stop(createObject)

// 	// create first object

// 	response, err := context.RequestFuture(createObject, &CreateObjectRequest{Name: "test", Description: "test object"}, 1*time.Second).Result()
// 	require.Nil(t, err)

// 	createObjectResponse, ok := response.(*CreateObjectResponse)
// 	require.True(t, ok)
// 	require.Nil(t, createObjectResponse.Error)

// 	// try to create second object w/ same name

// 	response, err = context.RequestFuture(createObject, &CreateObjectRequest{Name: "test", Description: "another test"}, 1*time.Second).Result()
// 	require.Nil(t, err)

// 	createObjectResponse, ok = response.(*CreateObjectResponse)
// 	require.True(t, ok)
// 	assert.NotNil(t, createObjectResponse.Error)
// 	assert.Nil(t, createObjectResponse.Object)
// }

// func TestInventoryActor_DropPickupObject(t *testing.T) {
// 	net := network.NewLocalNetwork()

// 	playerChainTree, err := net.CreateNamedChainTree("player")
// 	require.Nil(t, err)
// 	testPlayer := NewPlayerTree(net, playerChainTree)

// 	inventory, err := rootCtx.SpawnNamed(NewInventoryActorProps(&InventoryActorConfig{
// 		Player:  testPlayer,
// 		Network: net,
// 	}), "testDropObject")
// 	require.Nil(t, err)
// 	defer rootCtx.Stop(inventory)

// 	homeTree, err := createHome(net)
// 	require.Nil(t, err)
// 	c := new(navigator.Cursor).SetLocation(0, 0).SetChainTree(homeTree)
// 	loc, err := c.GetLocation()
// 	require.Nil(t, err)

// 	homeActor, err := rootCtx.SpawnNamed(NewLandActorProps(&LandActorConfig{
// 		Did:     homeTree.MustId(),
// 		Network: net,
// 	}), "home")
// 	require.Nil(t, err)
// 	defer rootCtx.Stop(homeActor)

// 	response, err := rootCtx.RequestFuture(inventory, &CreateObjectRequest{Name: "test", Description: "test object"}, 1*time.Second).Result()
// 	require.Nil(t, err)

// 	createObjectResponse, ok := response.(*CreateObjectResponse)
// 	require.True(t, ok)
// 	require.Nil(t, createObjectResponse.Error)

// 	response, err = rootCtx.RequestFuture(inventory, &DropObjectRequest{Name: "test", Location: loc}, 1*time.Second).Result()
// 	require.Nil(t, err)

// 	dropObjectResponse, ok := response.(*DropObjectResponse)
// 	require.True(t, ok)
// 	require.Nil(t, dropObjectResponse.Error)

// 	objectsPath, _ := consensus.DecodePath(ObjectsPath)
// 	objectPath := append(objectsPath, "test")
// 	playerObjNode, remainingPath, err := testPlayer.ChainTree().ChainTree.Dag.Resolve(append([]string{"tree", "data"}, objectPath...))
// 	require.Nil(t, err)
// 	require.Nil(t, playerObjNode)
// 	require.Equal(t, remainingPath, []string{"test"})

// 	// Give time for location to pickup change and refresh
// 	time.Sleep(500 * time.Millisecond)
// 	homeTree, err = net.GetTree(c.Did())
// 	require.Nil(t, err)
// 	c.SetChainTree(homeTree)
// 	loc, err = c.GetLocation()
// 	require.Nil(t, err)

// 	response, err = rootCtx.RequestFuture(inventory, &PickupObjectRequest{Name: "test", Location: loc}, 1*time.Second).Result()
// 	require.Nil(t, err)

// 	playerObjNode, remainingPath, err = testPlayer.ChainTree().ChainTree.Dag.Resolve(append([]string{"tree", "data"}, objectPath...))
// 	require.Nil(t, err)
// 	require.Nil(t, remainingPath)
// 	require.Equal(t, playerObjNode.(string), createObjectResponse.Object.Did)
// }
