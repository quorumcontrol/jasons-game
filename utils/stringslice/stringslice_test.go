package stringslice

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var s = []string{"c", "b", "a", "c"}

func TestIndex(t *testing.T) {
	require.Equal(t, Index(s, "c"), 0)
	require.Equal(t, Index(s, "b"), 1)
	require.Equal(t, Index(s, "a"), 2)

	require.Equal(t, Index(s, "d"), -1)
}

func TestInclude(t *testing.T) {
	require.True(t, Include(s, "c"))
	require.True(t, Include(s, "b"))
	require.True(t, Include(s, "a"))
	require.False(t, Include(s, "d"))
}

func TestEqual(t *testing.T) {
	compare := make([]string, len(s))
	copy(compare, s)
	require.True(t, Equal(s, compare))

	Reverse(compare)
	require.True(t, Equal(s, compare))
}

func TestAny(t *testing.T) {
	require.True(t, Any(s, func(str string) bool {
		return str == "a"
	}))
	require.False(t, Any(s, func(str string) bool {
		return str == "d"
	}))
}

func TestAll(t *testing.T) {
	require.True(t, All(s, func(str string) bool {
		return true
	}))
	require.False(t, All(s, func(str string) bool {
		return str == "a"
	}))
	require.False(t, All(s, func(str string) bool {
		return str == "d"
	}))
}

func TestReverse(t *testing.T) {
	compare := make([]string, len(s))
	copy(compare, s)
	Reverse(compare)
	require.Equal(t, compare, []string{"c", "a", "b", "c"})
}
