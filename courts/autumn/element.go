package autumn

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
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
