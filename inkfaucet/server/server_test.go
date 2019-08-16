// +build integration

package server

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/jasons-game/devink/devink"
	ifconfig "github.com/quorumcontrol/jasons-game/inkfaucet/config"
	"github.com/quorumcontrol/jasons-game/inkfaucet/depositor"
	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
)

func TestInkRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	devInk, err := devink.NewSource(ctx, "/tmp/test-dev-ink", true)
	defer func() {
		err = os.RemoveAll("/tmp/test-dev-ink")
		if err != nil {
			log.Warning("error removing /tmp/test-dev-ink dir")
		}
	}()
	require.Nil(t, err)

	inkFaucetKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	inkFaucetDID := KeyToDID(inkFaucetKey)

	cfg := ifconfig.InkFaucetConfig{
		Local:        true,
		InkOwnerDID:  devInk.ChainTree.MustId(),
		InkFaucetDID: inkFaucetDID,
		PrivateKey:   inkFaucetKey,
	}

	err = devInk.EnsureToken(ctx)
	require.Nil(t, err)

	err = devInk.EnsureBalance(ctx, 1000)
	require.Nil(t, err)

	server, err := New(ctx, cfg)
	require.Nil(t, err)

	tokenName := consensus.TokenName{ChainTreeDID: devInk.ChainTree.MustId(), LocalName: "ink"}

	assert.Equal(t, ctx, server.parentCtx)
	assert.Equal(t, tokenName.String(), server.tokenName.String())

	tokenSend, err := devInk.SendInk(ctx, server.InkFaucetDID(), 10)
	require.Nil(t, err)

	dep, err := depositor.New(ctx, cfg)
	require.Nil(t, err)

	err = dep.Deposit(tokenSend)
	require.Nil(t, err)

	err = server.Start(false)
	require.Nil(t, err)

	inkRecipient, err := devInk.Net.CreateNamedChainTree("ink-recipient")
	require.Nil(t, err)

	rootContext := actor.EmptyRootContext

	req := rootContext.RequestFuture(server.handler, &inkfaucet.InkRequest{Amount: 1, DestinationChainId: inkRecipient.MustId()}, 10*time.Second)

	uncastResp, err := req.Result()
	require.Nil(t, err)

	resp, ok := uncastResp.(*inkfaucet.InkResponse)
	require.True(t, ok)

	require.NotEmpty(t, resp.Token)

	var tokenPayload transactions.TokenPayload
	err = proto.Unmarshal(resp.Token, &tokenPayload)
	require.Nil(t, err)

	err = devInk.Net.ReceiveInk(inkRecipient, &tokenPayload)
	require.Nil(t, err)

	recipientTree, err := inkRecipient.ChainTree.Tree(ctx)
	require.Nil(t, err)

	recipientLedger := consensus.NewTreeLedger(recipientTree, &tokenName)

	recipientTokenExists, err := recipientLedger.TokenExists()
	require.Nil(t, err)
	assert.True(t, recipientTokenExists)

	recipientTokenBalance, err := recipientLedger.Balance()
	require.Nil(t, err)
	assert.Equal(t, uint64(1), recipientTokenBalance)
}
