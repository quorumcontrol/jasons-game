package game

import (
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
	tree    *consensus.SignedChainTree
	player  *jasonsgame.Player
	network network.Network
	did     string
	tip     cid.Cid
}

func NewPlayerTree(net network.Network, tree *consensus.SignedChainTree) *PlayerTree {
	pt := &PlayerTree{
		network: net,
	}
	pt.setTree(tree)
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

func (pt *PlayerTree) SetPlayer(p *jasonsgame.Player) error {
	tree, err := pt.network.UpdateChainTree(pt.tree, playerTreePath, p)
	if err != nil {
		return errors.Wrap(err, "error updating tree")
	}
	pt.setTree(tree)
	pt.player = p
	return nil
}

func (pt *PlayerTree) SetName(name string) error {
	p, err := pt.Player()
	if err != nil {
		return errors.Wrap(err, "error getting player")
	}
	p.Name = name
	return pt.SetPlayer(p)
}

func (pt *PlayerTree) setTree(tree *consensus.SignedChainTree) {
	pt.tree = tree
	pt.did = tree.MustId()
}

func (pt *PlayerTree) refresh() error {
	ret, remain, err := pt.tree.ChainTree.Dag.Resolve(strings.Split("tree/data/"+playerTreePath, "/"))
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
