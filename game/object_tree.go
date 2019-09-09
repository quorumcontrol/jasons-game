package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/game/trees"
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

	objectChainTree, err := net.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating object chaintree")
	}

	return CreateObjectOnTree(net, name, objectChainTree)
}

func CreateObjectOnTree(net network.Network, name string, tree *consensus.SignedChainTree) (*ObjectTree, error) {
	obj := NewObjectTree(net, tree)

	err := obj.SetName(name)
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

func (o *ObjectTree) ChainTree() *consensus.SignedChainTree {
	return o.tree
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

func (o *ObjectTree) AddDefaultInscriptionInteractions() error {
	name, err := o.GetName()
	if err != nil {
		return err
	}

	err = o.AddInteraction(&SetTreeValueInteraction{
		Command:  "inscribe object " + name,
		Did:      o.MustId(),
		Path:     "inscriptions",
		Multiple: true,
	})
	if err != nil {
		return errors.Wrap(err, "error adding interactions to object")
	}

	err = o.AddInteraction(&GetTreeValueInteraction{
		Command: "read inscriptions on object " + name,
		Did:     o.MustId(),
		Path:    "inscriptions",
	})
	if err != nil {
		return errors.Wrap(err, "error adding interactions to object")
	}
	return nil
}

func (o *ObjectTree) IsOwnedBy(keyAddrs []string) (bool, error) {
	return trees.VerifyOwnership(context.Background(), o.tree.ChainTree, keyAddrs)
}

func (o *ObjectTree) ChangeChainTreeOwner(newKeys []string) error {
	newTree, err := o.network.ChangeChainTreeOwner(o.tree, newKeys)
	if err != nil {
		return err
	}
	o.tree = newTree
	return nil
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

func (o *ObjectTree) UpdatePath(path []string, val interface{}) error {
	return o.updatePath(path, val)
}

func (o *ObjectTree) updatePath(path []string, val interface{}) error {
	newTree, err := o.network.UpdateChainTree(o.tree, strings.Join(append([]string{"jasons-game"}, path...), "/"), val)
	if err != nil {
		return err
	}
	o.tree = newTree
	return nil
}

func (o *ObjectTree) GetPath(path []string) (interface{}, error) {
	return o.getPath(path)
}

func (o *ObjectTree) getPath(path []string) (interface{}, error) {
	ctx := context.TODO()
	resp, _, err := o.tree.ChainTree.Dag.Resolve(ctx, append([]string{"tree", "data", "jasons-game"}, path...))
	if err != nil {
		return nil, fmt.Errorf("error resolving %v on object: %v", strings.Join(path, "/"), resp)
	}
	return resp, nil
}
