package game

import (
	"fmt"
	"strings"

	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/mitchellh/mapstructure"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func init() {
	cbor.RegisterCborType(Interaction{})
	typecaster.AddType(Interaction{})
}

type LocationTree struct {
	tree    *consensus.SignedChainTree
	network network.Network
}

type Interaction struct {
	Command string
	Action  string
	Args    map[string]string
}

func NewLocationTree(net network.Network, tree *consensus.SignedChainTree) *LocationTree {
	return &LocationTree{
		tree:    tree,
		network: net,
	}
}

func (l *LocationTree) Id() (string, error) {
	return l.tree.Id()
}

func (l *LocationTree) MustId() string {
	return l.tree.MustId()
}

func (l *LocationTree) Tip() cid.Cid {
	return l.tree.Tip()
}

func (l *LocationTree) GetDescription() (string, error) {
	val, err := l.getPath([]string{"description"})
	return val.(string), err
}

func (l *LocationTree) SetDescription(description string) error {
	return l.updatePath([]string{"description"}, description)
}

func (l *LocationTree) AddInteraction(i *Interaction) error {
	resp, err := l.GetInteraction(i.Command)
	if err != nil {
		return err
	}
	fmt.Printf("Response is %v", resp)
	if resp != nil {
		return fmt.Errorf("interaction %v already exists", i.Command)
	}
	fmt.Printf("Adding interaction %v", i)
	return l.updatePath([]string{"interactions", i.Command}, i)
}

func (l *LocationTree) GetInteraction(command string) (*Interaction, error) {
	val, err := l.getPath([]string{"interactions", command})
	if err != nil || val == nil {
		return nil, err
	}

	var interaction Interaction

	err = mapstructure.Decode(val, &interaction)
	if err != nil {
		return nil, err
	}

	return &interaction, nil
}

func (l *LocationTree) updatePath(path []string, val interface{}) error {
	newTree, err := l.network.UpdateChainTree(l.tree, strings.Join(append([]string{"jasons-game"}, path...), "/"), val)
	if err != nil {
		return err
	}
	l.tree = newTree
	return nil
}

func (l *LocationTree) getPath(path []string) (interface{}, error) {
	resp, _, err := l.tree.ChainTree.Dag.Resolve(append([]string{"tree", "data", "jasons-game"}, path...))
	if err != nil {
		return nil, fmt.Errorf("error resolving %v on location: %v", strings.Join(path, "/"), resp)
	}
	return resp, nil
}
