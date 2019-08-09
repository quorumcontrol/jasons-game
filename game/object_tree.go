package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/network"
)

type Object struct {
	Did string
}

type ObjectTree struct {
	tree    *consensus.SignedChainTree
	network network.Network
	withInteractions
}

func NewObjectTree(net network.Network, tree *consensus.SignedChainTree) *ObjectTree {
	return &ObjectTree{
		tree:    tree,
		network: net,
	}
}

func FindObjectTree(net network.Network, did string) (*ObjectTree, error) {
	objTree, err := net.GetTree(did)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error fetching object %v", did))
	}

	return NewObjectTree(net, objTree), nil
}

func CreateObjectTree(net network.Network, name string) (*ObjectTree, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required to create an object")
	}

	chainTreeName := fmt.Sprintf("object:%s", name)

	existingObj, err := net.GetChainTreeByName(chainTreeName)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error checking for existing chaintree; object name: %s", name))
	}
	if existingObj != nil {
		return nil, fmt.Errorf("object with name %s already exists; names must be unique", name)
	}

	objectChainTree, err := net.CreateNamedChainTree(chainTreeName)
	if err != nil {
		return nil, errors.Wrap(err, "error creating object chaintree")
	}

	obj := NewObjectTree(net, objectChainTree)

	err = obj.SetName(name)
	if err != nil {
		return nil, errors.Wrap(err, "error setting name of new object")
	}

	err = obj.AddInteraction(&DropObjectInteraction{
		Command: "drop object " + name,
		Did:     obj.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interactions to new object")
	}

	err = obj.AddInteraction(&PickUpObjectInteraction{
		Command: "pick up object " + name,
		Did:     obj.MustId(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interactions to new object")
	}

	err = obj.AddInteraction(&GetTreeValueInteraction{
		Command: "examine object " + name,
		Did:     obj.MustId(),
		Path:    "description",
	})
	if err != nil {
		return nil, errors.Wrap(err, "error adding interactions to new object")
	}

	return obj, nil
}

func (o *ObjectTree) Id() (string, error) {
	return o.tree.Id()
}

func (o *ObjectTree) MustId() string {
	return o.tree.MustId()
}

func (o *ObjectTree) Tip() cid.Cid {
	return o.tree.Tip()
}

func (o *ObjectTree) SetName(name string) error {
	return o.updatePath([]string{"name"}, name)
}

func (o *ObjectTree) GetName() (string, error) {
	return o.getProp("name")
}

func (o *ObjectTree) SetDescription(desc string) error {
	return o.updatePath([]string{"description"}, desc)
}

func (o *ObjectTree) GetDescription() (string, error) {
	return o.getProp("description")
}

func (o *ObjectTree) AddInteraction(i Interaction) error {
	return o.addInteractionToTree(o, i)
}

func (o *ObjectTree) InteractionsList() ([]Interaction, error) {
	return o.interactionsListFromTree(o)
}

func (o *ObjectTree) IsOwnedBy(keyAddrs []string) (bool, error) {
	ctx := context.TODO()
	authsUncasted, remainingPath, err := o.tree.ChainTree.Dag.Resolve(ctx, strings.Split("tree/"+consensus.TreePathForAuthentications, "/"))
	if err != nil {
		return false, err
	}
	if len(remainingPath) > 0 {
		return false, fmt.Errorf("error resolving object: path elements remaining: %v", remainingPath)
	}

	for _, storedAddr := range authsUncasted.([]interface{}) {
		found := false
		for _, check := range keyAddrs {
			found = found || (storedAddr.(string) == check)
		}
		if !found {
			return false, nil
		}
	}

	return true, nil
}

func (o *ObjectTree) getProp(prop string) (string, error) {
	uncastVal, err := o.getPath([]string{prop})
	if err != nil {
		return "", err
	}

	val, ok := uncastVal.(string)
	if !ok {
		return "", fmt.Errorf("error casting %s to string; type is %T", prop, uncastVal)
	}

	return val, nil
}

func (o *ObjectTree) updatePath(path []string, val interface{}) error {
	newTree, err := o.network.UpdateChainTree(o.tree, strings.Join(append([]string{"jasons-game"}, path...), "/"), val)
	if err != nil {
		return err
	}
	o.tree = newTree
	return nil
}

func (o *ObjectTree) getPath(path []string) (interface{}, error) {
	ctx := context.TODO()
	resp, _, err := o.tree.ChainTree.Dag.Resolve(ctx, append([]string{"tree", "data", "jasons-game"}, path...))
	if err != nil {
		return nil, fmt.Errorf("error resolving %v on object: %v", strings.Join(path, "/"), resp)
	}
	return resp, nil
}
