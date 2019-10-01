package benchmark

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/courts/autumn"
	"github.com/quorumcontrol/jasons-game/courts/config"
)

func autumConfig() (*autumn.AutumnConfig, error) {
	cfg := &autumn.AutumnConfig{}
	cfgPath, err := filepath.Abs("./courts/yml/autumn/config.yml")
	if err != nil {
		return nil, err
	}
	log.Debug("autumn court config path:", cfgPath)
	err = config.ReadYaml(cfgPath, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func combineElements(client *autumn.MockElementClient, combineIds []int, resultId int) error {
	for _, id := range combineIds {
		log.Debugf("sending element %d", id)
		err := client.Send(id)
		if err != nil {
			return err
		}
		if !client.HasBowl() {
			return errors.New("client.HasBowl() returned false")
		}
	}

	time.Sleep(100 * time.Millisecond)

	msg, err := client.PickupBowl()
	if err != nil {
		return err
	}
	if msg.Error != "" {
		return errors.New(msg.Error)
	}

	if !client.HasElement(resultId) {
		return fmt.Errorf("expected element id %d but client.HasElement returned false", resultId)
	}

	return nil
}

func (tb *TransactionsBenchmark) combineWeaverElementsOnTrees(cfg *autumn.AutumnConfig, weaverTree, playerTree *consensus.SignedChainTree) error {
	weaver, err := autumn.NewElementCombinerHandler(&autumn.ElementCombinerHandlerConfig{
		Name:         "weaver",
		Network:      tb.net,
		Location:     weaverTree.MustId(),
		Elements:     cfg.Elements,
		Combinations: cfg.Weaver,
	})
	if err != nil {
		return err
	}

	weaverClient := &autumn.MockElementClient{
		Net:     tb.net,
		H:       weaver,
		Player:  playerTree.MustId(),
		Service: weaverTree.MustId(),
	}

	return combineElements(weaverClient, []int{100, 200}, 300)
}

func (tb *TransactionsBenchmark) combineWeaverElements() error {
	cfg, err := autumConfig()
	if err != nil {
		return err
	}

	weaverTree, err := tb.net.CreateChainTree()
	if err != nil {
		return err
	}

	playerTree, err := tb.net.CreateLocalChainTree("player")
	if err != nil {
		return err
	}

	return tb.combineWeaverElementsOnTrees(cfg, weaverTree, playerTree)
}

// NB: Doesn't work currently b/c element 300 is test only
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
// 	// do the weaver thing first to get element 300
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
// 		Service: binderTree.MustId(),
// 	}
//
// 	return combineElements(binderClient, []int{100, 200, 300}, 600)
// }
