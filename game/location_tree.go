package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var portalPath = []string{"portal"}

type LocationTree struct {
	tree    *consensus.SignedChainTree
	network network.Network
	withInteractions
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

func (l *LocationTree) AddInteraction(i Interaction) error {
	return l.addInteractionToTree(l, i)
}

func (l *LocationTree) InteractionsList() ([]Interaction, error) {
	return l.interactionsListFromTree(l)
}

func (l *LocationTree) SetHandler(handlerDid string) error {
	locationAuths, err := l.tree.Authentications()
	if err != nil {
		return errors.Wrap(err, "error fetching location auths")
	}

	handlerTree, err := l.network.GetTree(handlerDid)
	if err != nil {
		return errors.Wrap(err, "error fetching handler tree")
	}

	handlerAuths, err := handlerTree.Authentications()
	if err != nil {
		return errors.Wrap(err, "error fetching handler auths")
	}

	newTree, err := l.network.UpdateChainTree(l.tree, "jasons-game-handler", handlerDid)
	if err != nil {
		return errors.Wrap(err, "error setting new handler attr")
	}
	l.tree = newTree

	newTree, err = l.network.ChangeChainTreeOwner(l.tree, append(locationAuths, handlerAuths...))
	if err != nil {
		return errors.Wrap(err, "error setting new handler auths")
	}
	l.tree = newTree

	return nil
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

func (l *LocationTree) DeletePortal() error {
	currentPortal, err := l.GetPortal()

	if err != nil {
		return fmt.Errorf("error fetching portals: %v", err)
	}

	if currentPortal == nil {
		return fmt.Errorf("error, no portal to delete")
	}

	return l.updatePath(portalPath, nil)
}

func (l *LocationTree) GetPortal() (*jasonsgame.Portal, error) {
	portal, err := l.getPath(portalPath)
	if err != nil {
		return nil, fmt.Errorf("error fetching portal: %v", err)
	}

	if portal == nil {
		return nil, nil
	}

	castedPortal := new(jasonsgame.Portal)
	err = typecaster.ToType(portal, castedPortal)
	if err != nil {
		return nil, errors.Wrap(err, "error casting portal")
	}

	return castedPortal, nil
}

func (l *LocationTree) IsOwnedBy(keyAddrs []string) (bool, error) {
	return trees.VerifyOwnership(context.Background(), l.tree.ChainTree, keyAddrs)
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
	ctx := context.TODO()
	resp, _, err := l.tree.ChainTree.Dag.Resolve(ctx, append([]string{"tree", "data", "jasons-game"}, path...))
	if err != nil {
		return nil, fmt.Errorf("error resolving %v on location: %v", strings.Join(path, "/"), resp)
	}
	return resp, nil
}
