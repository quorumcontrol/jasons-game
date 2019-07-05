package trees

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

const ObjectsPath = "jasons-game/inventory"

type InventoryTree struct {
	tree    *consensus.SignedChainTree
	network network.Network
}

func NewInventoryTree(net network.Network, tree *consensus.SignedChainTree) *InventoryTree {
	return &InventoryTree{
		tree:    tree,
		network: net,
	}
}

func FindInventoryTree(net network.Network, did string) (*InventoryTree, error) {
	objTree, err := net.GetTree(did)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error fetching inventory for %v", did))
	}

	return NewInventoryTree(net, objTree), nil
}

func (t *InventoryTree) MustId() string {
	return t.tree.MustId()
}

func (t *InventoryTree) BroadcastTopic() []byte {
	return t.network.Community().TopicFor(t.tree.MustId() + "/inventory")
}

func (t *InventoryTree) Exists(did string) (bool, error) {
	allObjects, err := t.All()

	if err != nil {
		return false, err
	}

	_, ok := allObjects[did]
	return ok, nil
}

func (t *InventoryTree) All() (map[string]string, error) {
	ctx := context.TODO()

	resolveObjectsPath, _ := consensus.DecodePath(fmt.Sprintf("tree/data/%s", ObjectsPath))

	objectsUncasted, _, err := t.tree.ChainTree.Dag.Resolve(ctx, resolveObjectsPath)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching inventory")
	}

	if objectsUncasted == nil {
		return make(map[string]string), nil
	}

	objects := make(map[string]string, len(objectsUncasted.(map[string]interface{})))
	for name, did := range objectsUncasted.(map[string]interface{}) {
		objects[did.(string)] = name
	}
	return objects, nil
}

func (t *InventoryTree) Remove(did string) error {
	allObjects, err := t.All()
	if err != nil {
		return err
	}

	_, ok := allObjects[did]
	if !ok {
		return nil
	}

	delete(allObjects, did)
	err = t.updateObjects(allObjects)
	return err
}

func (t *InventoryTree) Add(did string) error {
	ctx := context.TODO()

	allObjects, err := t.All()
	if err != nil {
		return err
	}

	_, ok := allObjects[did]
	if ok {
		return nil
	}

	objectTree, err := t.network.GetTree(did)
	if err != nil {
		return err
	}

	uncastObjectName, _, err := objectTree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "name"})
	if err != nil {
		return err
	}

	name, ok := uncastObjectName.(string)
	if !ok {
		return fmt.Errorf("error casting name to string; type is %T", uncastObjectName)
	}

	allObjects[did] = name
	err = t.updateObjects(allObjects)
	return err
}

func (t *InventoryTree) Authentications() ([]string, error) {
	return t.tree.Authentications()
}

func (t *InventoryTree) IsOwnedBy(keyAddrs []string) (bool, error) {
	auths, err := t.Authentications()
	if err != nil {
		return false, err
	}

	for _, storedAddr := range auths {
		found := false
		for _, check := range keyAddrs {
			found = found || (storedAddr == check)
		}
		if !found {
			return false, nil
		}
	}

	return true, nil
}

func (t *InventoryTree) updateObjects(objects map[string]string) error {
	reversed := make(map[string]string, len(objects))

	for did, name := range objects {
		reversed[name] = did
	}

	newTree, err := t.network.UpdateChainTree(t.tree, ObjectsPath, reversed)
	if err != nil {
		return errors.Wrap(err, "error updating objects in inventory")
	}
	t.tree = newTree
	return nil
}
