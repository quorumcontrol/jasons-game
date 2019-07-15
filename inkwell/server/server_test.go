// +build integration

package server

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/jasons-game/devink/devink"
	iwconfig "github.com/quorumcontrol/jasons-game/inkwell/config"
	"github.com/quorumcontrol/jasons-game/inkwell/depositor"
	"github.com/quorumcontrol/jasons-game/inkwell/inkwell"
)

func TestInkRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := iwconfig.InkwellConfig{
		Local: true,
		S3Bucket: "test",
	}

	devInk, err := devink.NewSource(ctx, "/tmp/test-dev-ink", true)
	defer func() {
		err = os.RemoveAll("/tmp/test-dev-ink")
		if err != nil {
			log.Warning("error removing /tmp/test-dev-ink dir")
		}
	}()
	require.Nil(t, err)

	err = os.Setenv("INK_DID", devInk.ChainTree.MustId())
	require.Nil(t, err)

	err = devInk.EnsureToken(ctx)
	require.Nil(t, err)

	err = devInk.EnsureBalance(ctx, 1000)
	require.Nil(t, err)

	server, err := New(ctx, cfg)
	require.Nil(t, err)

	tokenName := consensus.TokenName{ChainTreeDID: devInk.ChainTree.MustId(), LocalName: "ink"}

	assert.Equal(t, ctx, server.parentCtx)
	assert.Equal(t, tokenName.String(), server.tokenName.String())
	// TODO: Assert some mo' thangs.

	tokenSend, err := devInk.SendInk(ctx, server.InkwellDID(), 10)
	require.Nil(t, err)

	dep, err := depositor.New(ctx, cfg)
	require.Nil(t, err)

	fmt.Printf("tokenSend: %+v\n", tokenSend)
	err = dep.Deposit(tokenSend)
	require.Nil(t, err)

	err = server.Start()
	require.Nil(t, err)

	rootContext := actor.EmptyRootContext

	req := rootContext.RequestFuture(server.handler, inkwell.InkRequest{Amount: 1, DestinationChainId: "other-did"}, 1 * time.Second)

	uncastResp, err := req.Result()
	require.Nil(t, err)

	resp, ok := uncastResp.(*inkwell.InkResponse)
	require.True(t, ok)

	assert.Equal(t, "foo", resp.Token)
}
