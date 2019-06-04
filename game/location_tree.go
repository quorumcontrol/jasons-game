package game

import (
	"fmt"
	"strings"

	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/mitchellh/mapstructure"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var portalPath = []string{"portal"}

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
	if err != nil || val == nil {
		return "", err
	}
	return val.(string), err
}

func (l *LocationTree) SetDescription(description string) error {
	return l.updatePath([]string{"description"}, description)
}

func (l *LocationTree) AddInteraction(i *Interaction) error {
	resp, err := l.GetInteractionRequest(i.Command)
	if err != nil {
		return err
	}
	if resp != nil {
		return fmt.Errorf("interaction %v already exists", i.Command)
	}
	return l.updatePath([]string{"interactions", i.Command}, i)
}

func (l *LocationTree) GetInteractionRequest(command string) (*Interaction, error) {
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

func (l *LocationTree) BuildPortal(toDid string) error {
	currentPortal, err := l.GetPortal()

	if err != nil {
		return fmt.Errorf("error fetching portals: %v", err)
	}

	if currentPortal != nil {
		return fmt.Errorf("error, portal already exists")
	}

	portal := &jasonsgame.Portal{To: toDid}
	return l.updatePath(portalPath, portal)
}

func (l *LocationTree) GetPortal() (*jasonsgame.Portal, error) {
	portal, err := l.getPath(portalPath)
	if err != nil {
		return nil, fmt.Errorf("error fetching portal: %v", err)
	}

	if portal == nil {
		return nil, nil
	}

	var castedPortal *jasonsgame.Portal
	err = mapstructure.Decode(portal, &castedPortal)
	if err != nil {
		return nil, fmt.Errorf("error casting portal: %v", err)
	}

	return castedPortal, nil
}

func (l *LocationTree) IsOwnedBy(keys []string) (bool, error) {
	authsUncasted, remainingPath, err := l.tree.ChainTree.Dag.Resolve(strings.Split("tree/"+consensus.TreePathForAuthentications, "/"))
	if err != nil {
		return false, err
	}
	if len(remainingPath) > 0 {
		return false, fmt.Errorf("error resolving tree: path elements remaining: %v", remainingPath)
	}

	for _, storedKey := range authsUncasted.([]interface{}) {
		found := false
		for _, checkKey := range keys {
			found = found || (storedKey.(string) == checkKey)
		}
		if !found {
			return false, nil
		}
	}

	return true, nil
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
