package navigator

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/storage"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/stretchr/testify/require"
)

func TestCursor(t *testing.T) {
	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())

	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	tree, err := consensus.NewSignedChainTree(key.PublicKey, store)
	require.Nil(t, err)

	updated, err := tree.ChainTree.Dag.SetAsLink([]string{"tree", "data", "jasons-game", "0", "0"}, &Location{Description: "hi"})
	require.Nil(t, err)
	require.NotNil(t, updated)

	updated, err = updated.SetAsLink([]string{"tree", "data", "jasons-game", "0", "1"}, &Location{Description: "north"})
	require.Nil(t, err)
	require.NotNil(t, updated)
	tree.ChainTree.Dag = updated

	cursor := new(Cursor)
	output, err := cursor.SetLocation(0, 0).SetChainTree(tree).GetLocation()
	require.Nil(t, err)
	require.Equal(t,
		&Location{
			Description: "hi",
			Did:         tree.MustId(),
			X:           0,
			Y:           0,
		},
		output)

	cursor.North()
	output, err = cursor.GetLocation()
	require.Nil(t, err)
	require.Equal(t,
		&Location{
			Description: "north",
			Did:         tree.MustId(),
			X:           0,
			Y:           1,
		},
		output)

	cursor.South()
	output, err = cursor.GetLocation()
	require.Nil(t, err)
	require.Equal(t,
		&Location{
			Description: "hi",
			Did:         tree.MustId(),
			X:           0,
			Y:           0,
		},
		output)
}
