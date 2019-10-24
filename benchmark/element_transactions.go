package benchmark

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/quorumcontrol/jasons-game/courts/autumn"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers/broadcast"
	"github.com/quorumcontrol/messages/build/go/signatures"
)

const weaverServiceDid = "did:tupelo:0x55e6099c0a47c8516e72e402B10b9e02601ADa6C"

// const binderServiceDid = "did:tupelo:0x99bcF7ECC24F028dB1080a9d76b20D08526327BF"

func combineElements(client *autumn.MockElementClient, combineIds []int, resultId int) error {
	locationTree, err := client.Net.GetTree(client.Location)
	if err != nil {
		return err
	}

	startingHeight := trees.MustHeight(context.Background(), locationTree.ChainTree)
	firstExpectedHeight := startingHeight + 1
	lastExpectedHeight := startingHeight + uint64(len(combineIds)) + 1 // +1 for bowl

	futures := make(map[uint64]*actor.Future)

	for i := firstExpectedHeight; i <= lastExpectedHeight; i++ {
		futures[uint64(i)] = actor.NewFuture(300 * time.Second)
	}

	pid := actor.EmptyRootContext.Spawn(actor.PropsFromFunc(func(actorCtx actor.Context) {
		switch msg := actorCtx.Message().(type) {
		case *actor.Started:
			actorCtx.Spawn(client.Net.NewCurrentStateSubscriptionProps(locationTree.MustId()))
		case *signatures.CurrentState:
			if future, ok := futures[msg.Signature.Height]; ok {
				actorCtx.Send(future.PID(), msg)
			}
		}
	}))
	defer actor.EmptyRootContext.Stop(pid)

	currentExpectedHeight := firstExpectedHeight

	time.Sleep(500 * time.Millisecond)

	for _, id := range combineIds {
		log.Debugf("sending element %d", id)
		err := client.Send(id)
		if err != nil {
			return err
		}
		_, err = futures[currentExpectedHeight].Result()
		if err != nil {
			return fmt.Errorf("error waiting for element")
		}
		currentExpectedHeight++
	}

	msg, err := client.PickupBowl()
	if err != nil {
		return err
	}
	if msg.Error != "" {
		return errors.New(msg.Error)
	}
	_, err = futures[lastExpectedHeight].Result()
	if err != nil {
		return fmt.Errorf("error waiting for bowl transfer")
	}
	if !client.HasElement(resultId) {
		return fmt.Errorf("expected element id %d but client.HasElement returned false", resultId)
	}
	return nil
}

func (tb *TransactionsBenchmark) combineWeaverElements() error {
	playerTree, err := tb.net.CreateLocalChainTree("player")
	if err != nil {
		return err
	}

	locationTree, err := tb.net.CreateChainTree()
	if err != nil {
		return err
	}

	location := game.NewLocationTree(tb.net, locationTree)

	err = location.SetHandler(weaverServiceDid)
	if err != nil {
		return err
	}

	handler := broadcast.NewTopicBroadcastHandler(tb.net, tb.net.Community().TopicFor(weaverServiceDid))

	weaverClient := &autumn.MockElementClient{
		Net:      tb.net,
		H:        handler,
		Player:   playerTree.MustId(),
		Location: location.MustId(),
	}

	return combineElements(weaverClient, []int{24, 26}, 25)
}

// NB: Doesn't work currently
// func (tb *TransactionsBenchmark) combineBinderElements() error {
// 	cfg, err := autumConfig()
// 	if err != nil {
// 		return err
// 	}
//
// 	playerTree, err := tb.net.CreateLocalChainTree("player")
// 	if err != nil {
// 		return err
// 	}
//
// 	// do the weaver thing first to get needed element
// 	weaverTree, err := tb.net.CreateChainTree()
// 	if err != nil {
// 		return err
// 	}
//
// 	err = tb.combineWeaverElementsOnTrees(cfg, weaverTree, playerTree)
// 	if err != nil {
// 		return err
// 	}
//
// 	binderTree, err := tb.net.CreateChainTree()
// 	if err != nil {
// 		return err
// 	}
//
// 	binder, err := autumn.NewElementCombinerHandler(&autumn.ElementCombinerHandlerConfig{
// 		Name:         "binder",
// 		Network:      tb.net,
// 		Location:     binderTree.MustId(),
// 		Elements:     cfg.Elements,
// 		Combinations: cfg.Binder,
// 	})
// 	if err != nil {
// 		return err
// 	}
//
// 	binderClient := &autumn.MockElementClient{
// 		Net:     tb.net,
// 		H:       binder,
// 		Player:  playerTree.MustId(),
// 		Location: binderTree.MustId(),
// 	}
//
// 	return combineElements(binderClient, cfg.Binder[0].From, cfg.Binder[0].To)
// }
