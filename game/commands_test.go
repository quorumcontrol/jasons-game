package game

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandList(t *testing.T) {
	comm, _ := defaultCommandList.findCommand("help")
	require.NotNil(t, comm)
}
