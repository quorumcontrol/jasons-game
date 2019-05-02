package game

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandList(t *testing.T) {
	comm, _ := defaultCommandList.findCommand("north")
	require.NotNil(t, comm)
}
