package autumn

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestElement(t *testing.T) {
	elements := map[int]string{
		1: "1",
		100: "64",
		1000: "3e8",
	}

	for elementID, elementSuffix := range elements {
		e := element{ ID: elementID }
		require.Equal(t, e.Name(), elementNamePrefix + elementSuffix)
	}

	for i := 1;  i<=103; i++ {
		e := element{ ID: i }
		require.Equal(t, i, elementNameToId(e.Name()))
	}
}