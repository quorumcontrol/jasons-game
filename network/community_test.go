package network

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/stretchr/testify/require"
)

func TestCommunity_TopicFor(t *testing.T) {
	ctx := context.Background()
	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	p2pHost, err := p2p.NewLibP2PHost(ctx, key, 0)
	require.Nil(t, err)
	community := NewJasonCommunity(ctx, key, p2pHost)

	aDid := "did:tupelo:0x634635264b61a9721A172807841b8B3a65ee9549"
	require.Equal(t, community.TopicFor(aDid), []byte(aDid))
}
