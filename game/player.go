package game

import (
	"fmt"
	"strings"

	cbor "github.com/ipfs/go-ipld-cbor"

	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
)

func init() {
	cbor.RegisterCborType(PlayerInfo{})
	typecaster.AddType(PlayerInfo{})
}

type Player struct {
	tree *consensus.SignedChainTree
}

type PlayerInfo struct {
	Name string
}

func (p *Player) GetInfo() (*PlayerInfo, error) {
	pth, remain, err := p.tree.ChainTree.Dag.Resolve(strings.Split("tree/data/jasons-game/player", "/"))
	if err != nil {
		return nil, fmt.Errorf("error resolving: %v", err)
	}
	if len(remain) > 0 {
		return nil, fmt.Errorf("error, path remaining: %v", remain)
	}

	pi := new(PlayerInfo)
	err = typecaster.ToType(pth, pi)
	if err != nil {
		return nil, fmt.Errorf("error casting: %v", err)
	}
	return pi, nil
}
