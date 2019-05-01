package game

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandList(t *testing.T) {
	comm, _ := defaultCommandList.findCommand("north")
	require.NotNil(t, comm)
}

func TestCommandMatches(t *testing.T) {
	comm := newCommand("test", "test")

	_, err := comm.allot.Match("test")
	require.Nil(t, err)
}
