package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var playerTreePath = "jasons-game/player"

type PlayerTree struct {
	tree         *consensus.SignedChainTree
	HomeLocation *LocationTree
	player       *jasonsgame.Player
	network      network.Network
	did          string
}

func NewPlayerTree(net network.Network, tree *consensus.SignedChainTree) *PlayerTree {
	pt := &PlayerTree{
		network: net,
	}
	pt.setTree(tree)

	homeTree, err := net.GetChainTreeByName("home")
	if err != nil {
		panic(err)
	}
	if homeTree == nil {
		pt.HomeLocation, err = createHome(net)
		if err != nil {
			log.Error("error creating home", err)
			panic(err)
		}
	} else {
		pt.HomeLocation = NewLocationTree(net, homeTree)
	}

	return pt
}

func (pt *PlayerTree) Did() string {
	return pt.did
}

func (pt *PlayerTree) Tip() cid.Cid {
	return pt.tree.Tip()
}

func (pt *PlayerTree) Player() (*jasonsgame.Player, error) {
	if pt.player == nil {
		err := pt.refresh()
		if err != nil {
			return nil, errors.Wrap(err, "error refreshing from tree")
		}
	}
	return pt.player, nil
}

func (pt *PlayerTree) Authentications() ([]string, error) {
	ctx := context.TODO()
	authsUncasted, remain, err := pt.tree.ChainTree.Dag.Resolve(ctx, strings.Split("tree/"+consensus.TreePathForAuthentications, "/"))
	if err != nil {
		return nil, errors.Wrap(err, "error resolving")
	}
	if len(remain) > 0 {
		return nil, fmt.Errorf("error, path remaining: %v", remain)
	}

	auths := make([]string, len(authsUncasted.([]interface{})))
	for k, v := range authsUncasted.([]interface{}) {
		auths[k] = v.(string)
	}

	return auths, nil
}

func (pt *PlayerTree) SetPlayer(p *jasonsgame.Player) error {
	tree, err := pt.network.UpdateChainTree(pt.tree, playerTreePath, p)
	if err != nil {
		return errors.Wrap(err, "error updating tree")
	}
	pt.setTree(tree)
	pt.player = p
	return nil
}

func (pt *PlayerTree) setTree(tree *consensus.SignedChainTree) {
	pt.tree = tree
	pt.did = tree.MustId()
}

func (pt *PlayerTree) refresh() error {
	ctx := context.TODO()
	ret, remain, err := pt.tree.ChainTree.Dag.Resolve(ctx, strings.Split("tree/data/"+playerTreePath, "/"))
	if err != nil {
		return errors.Wrap(err, "error resolving")
	}
	if len(remain) > 0 {
		return fmt.Errorf("error, path remaining: %v", remain)
	}

	p := new(jasonsgame.Player)
	err = typecaster.ToType(ret, p)
	if err != nil {
		return errors.Wrap(err, "error casting")
	}
	pt.player = p
	return nil
}

func (p *PlayerTree) ChainTree() *consensus.SignedChainTree {
	return p.tree
}

func (p *PlayerTree) SetChainTree(ct *consensus.SignedChainTree) {
	p.tree = ct
}

func GetPlayerTree(net network.Network) (*PlayerTree, error) {
	playerChain, err := net.GetChainTreeByName("player")
	if err != nil {
		return nil, err
	}

	if playerChain == nil {
		return nil, nil
	}

	return NewPlayerTree(net, playerChain), nil
}

func CreatePlayerTree(net network.Network, chainTreeId string) (*PlayerTree, error) {
	playerChain, err := net.GetTree(chainTreeId)
	if err != nil {
		return nil, err
	}
	if playerChain == nil {
		return nil, errors.Errorf("player chaintree %s was not found", chainTreeId)
	}

	playerTree := NewPlayerTree(net, playerChain)
	if err := playerTree.SetPlayer(&jasonsgame.Player{
		Name: fmt.Sprintf("newb (%s)", playerChain.MustId()),
	}); err != nil {
		return nil, err
	}

	return playerTree, nil
}
