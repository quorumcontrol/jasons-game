package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

type InteractionTree struct {
	tree    *consensus.SignedChainTree
	network network.Network
	withInteractions
}

func NewInteractionTree(net network.Network, tree *consensus.SignedChainTree) *InteractionTree {
	return &InteractionTree{
		tree:    tree,
		network: net,
	}
}

func (l *InteractionTree) Id() (string, error) {
	return l.tree.Id()
}

func (l *InteractionTree) MustId() string {
	return l.tree.MustId()
}

func (l *InteractionTree) Tip() cid.Cid {
	return l.tree.Tip()
}

func (l *InteractionTree) Tree() *consensus.SignedChainTree {
	return l.tree
}

func (l *InteractionTree) AddInteraction(i Interaction) error {
	return l.addInteractionToTree(l, i)
}

func (l *InteractionTree) InteractionsList() ([]Interaction, error) {
	return l.interactionsListFromTree(l)
}

func (l *InteractionTree) updatePath(path []string, val interface{}) error {
	newTree, err := l.network.UpdateChainTree(l.tree, strings.Join(append([]string{"jasons-game"}, path...), "/"), val)
	if err != nil {
		return err
	}
	l.tree = newTree
	return nil
}

func (l *InteractionTree) getPath(path []string) (interface{}, error) {
	ctx := context.TODO()
	resp, _, err := l.tree.ChainTree.Dag.Resolve(ctx, append([]string{"tree", "data", "jasons-game"}, path...))
	if err != nil {
		return nil, fmt.Errorf("error resolving %v on location: %v", strings.Join(path, "/"), resp)
	}
	return resp, nil
}
