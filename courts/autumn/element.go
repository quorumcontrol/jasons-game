package autumn

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
)

const elementNamePrefix = "element-"

type element struct {
	ID                   int    `yaml:"id"`
	Description          string `yaml:"description"`
	SkipOriginValidation bool   `yaml:"skip_origin_validation"`
}

func (e *element) Name() string {
	return elementNamePrefix + strconv.FormatInt(int64(e.ID), 16)
}

func validateElementOrigin(object *game.ObjectTree, auths []string) (bool, error) {
	ctx := context.Background()

	elementName, err := object.GetName()
	if err != nil {
		return false, fmt.Errorf("error checking origin of element: %v", err)
	}

	ownershipChanges, err := trees.OwnershipChanges(ctx, object.ChainTree().ChainTree)
	if err != nil {
		return false, fmt.Errorf("error checking origin of element: %v", err)
	}

	if len(ownershipChanges) < 2 {
		log.Debugf("validateElementOrigin: invalid ownership history, less than 2 ownership changes: obj=%s", object.MustId())
		return false, nil
	}

	beforeTransferOwnership := ownershipChanges[len(ownershipChanges)-2]

	originObject, err := object.AtTip(beforeTransferOwnership.Tip)
	if err != nil {
		return false, fmt.Errorf("error checking origin of element")
	}

	validOrigin, err := trees.VerifyOwnershipAt(ctx, originObject.ChainTree().ChainTree, 0, auths)
	if err != nil {
		return false, fmt.Errorf("error checking origin of element")
	}
	if !validOrigin {
		log.Debugf("validateElementOrigin: invalid ownership history, authentication at block 0 was not network key: obj=%s", object.MustId())
		return false, nil
	}

	originName, err := originObject.GetName()
	if err != nil {
		return false, fmt.Errorf("error checking origin of element")
	}

	if elementName != originName {
		log.Debugf("validateElementOrigin: invalid object, name was modified: obj=%s", object.MustId())
		return false, nil
	}

	return true, nil
}

func elementNameToId(name string) int {
	id, err := strconv.ParseInt(strings.TrimPrefix(name, elementNamePrefix), 16, 0)
	if err != nil {
		log.Errorf("decoding %s as hex errored: %v", name, err)
		return 0
	}
	return int(id)
}

type elementCombination struct {
	From []int `yaml:"from"`
	To   int   `yaml:"to"`
}

type elementCombinationMap map[string]int

func (m *elementCombinationMap) keyFromSlice(ints []int) string {
	sortedKeys := make(sort.IntSlice, len(ints))
	copy(sortedKeys, ints)
	sortedKeys.Sort()
	return fmt.Sprint(sortedKeys)
}

func (m elementCombinationMap) Find(ints []int) (int, bool) {
	key := m.keyFromSlice(ints)
	val, ok := m[key]
	return val, ok
}

func (m elementCombinationMap) Fill(combinations []*elementCombination) {
	for _, combination := range combinations {
		key := m.keyFromSlice(combination.From)
		m[key] = combination.To
	}
}
