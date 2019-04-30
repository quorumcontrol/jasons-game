// +build integration

package network

import(
	"github.com/stretchr/testify/require"
	"testing"
	"os"
	"path/filepath"
	"context"
)

func TestNewNetwork(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()

	testPath := "/tmp/test-new-network"

	os.MkdirAll(filepath.Join(testPath, "ipld"), 0755)
	os.MkdirAll(filepath.Join(testPath, "tupelo"), 0755)
	defer os.RemoveAll(testPath)


	group,err := setupNotaryGroup(ctx)
	require.Nil(t,err)

	_,err = NewRemoteNetwork(ctx, group, testPath)
	require.Nil(t,err)
}