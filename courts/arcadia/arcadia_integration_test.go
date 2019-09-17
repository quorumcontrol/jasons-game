// +build integration

package arcadia

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/stretchr/testify/require"
)

func setupTestArcadiaCourt(t *testing.T, ctx context.Context) (*ui.TestStream, *actor.PID) {
	group, err := network.SetupTupeloNotaryGroup(ctx, true)
	require.Nil(t, err)

	netKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	net, err := network.NewRemoteNetworkWithConfig(ctx, &network.RemoteNetworkConfig{
		NotaryGroup:   group,
		SigningKey:    netKey,
		NetworkKey:    netKey,
		KeyValueStore: config.MemoryDataStore(),
	})
	require.Nil(t, err)

	court := New(ctx, net, "../yml-test")
	court.Start()

	playerChain, err := net.CreateLocalChainTree("player")
	require.Nil(t, err)
	playerTree, err := game.CreatePlayerTree(net, playerChain.MustId())
	require.Nil(t, err)

	rootCtx := actor.EmptyRootContext
	stream := ui.NewTestStream(t)
	simulatedUI, err := rootCtx.SpawnNamed(ui.NewUIProps(stream, net), t.Name()+"-ui")
	require.Nil(t, err)

	gameCfg := &game.GameConfig{PlayerTree: playerTree, UiActor: simulatedUI, Network: net}
	gamePid, err := rootCtx.SpawnNamed(game.NewGameProps(gameCfg), t.Name()+"-game")
	require.Nil(t, err)

	time.Sleep(10 * time.Second)

	ids, err := court.court.Ids()
	require.Nil(t, err)
	require.NotNil(t, ids)
	locationIds := make(map[string]string)
	for name, did := range ids["Locations"].(map[string]interface{}) {
		locationIds[name] = did.(string)
	}
	require.Len(t, locationIds, 7)

	stream.ExpectMessage("built a portal", 3*time.Second)
	rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: fmt.Sprintf("build portal to %s", locationIds["main"])})
	stream.Wait()
	stream.ExpectMessage("main description", 3*time.Second)
	rootCtx.Send(gamePid, &jasonsgame.UserInput{Message: "go through portal"})
	stream.Wait()
	require.Nil(t, stream.ClearMessages())

	keys := []string{
		"0x6b181aa64c6afcac2fb711c1c2138ca7ea0b0688e30d2a9f0dbb2ce48ac61797",
		"0xd3140dbd1a73515afa2bac8212f8c7713b1493db19acfe22da3c7e66aa41aad2",
		"0x8412eef05c319b60333a0ea5c387b33e5b63e7fe6f67b83da5c98b02c7277873",
		"0x31d498c6a2f72e0ccfd63943098b7db2bc76ef8a462cc6e8665a40a55afaa119",
		"0xa605ff74334fb509c661120ddf64ce22a2c9f85ea595632856ce009ab67cff4d",
	}

	playerInventory := trees.NewInventoryTree(net, playerChain)
	for i := 1; i <= 5; i++ {
		ephemeralKey, err := crypto.GenerateKey()
		require.Nil(t, err)

		objTree, err := net.CreateChainTreeWithKey(ephemeralKey)
		require.Nil(t, err)

		key, err := crypto.ToECDSA(hexutil.MustDecode(keys[i-1]))
		require.Nil(t, err)

		// change to "service" owner
		serviceAddr := crypto.PubkeyToAddress(key.PublicKey).String()
		objTree, err = net.ChangeChainTreeOwnerWithKey(objTree, ephemeralKey, []string{serviceAddr})
		require.Nil(t, err)

		// required to be set by the "service"
		err = net.Tupelo.UpdateChainTree(objTree, key, "jasons-game/inscriptions/forged by", fmt.Sprintf("forged%d", i))
		require.Nil(t, err)

		// simulate "transfer" ownership transfer
		playerAddr := crypto.PubkeyToAddress(*net.PublicKey()).String()
		objTree, err = net.ChangeChainTreeOwnerWithKey(objTree, key, []string{serviceAddr, playerAddr})
		require.Nil(t, err)

		// full change to player owned
		objTree, err = net.ChangeChainTreeOwner(objTree, []string{playerAddr})
		require.Nil(t, err)

		obj, err := game.CreateObjectOnTree(net, fmt.Sprintf("artifact-%d", i), objTree)
		require.Nil(t, err)

		for _, inscriptionKey := range []string{"type", "material", "age", "weight"} {
			objTree, err = net.UpdateChainTree(obj.ChainTree(), fmt.Sprintf("jasons-game/inscriptions/%s", inscriptionKey), fmt.Sprintf("%s%d", inscriptionKey, i))
			require.Nil(t, err)
		}

		err = playerInventory.ForceAdd(objTree.MustId())
		require.Nil(t, err)
	}

	// create bad ownership fake artifact
	objTree, err := net.CreateChainTree()
	require.Nil(t, err)

	objTree, err = net.UpdateChainTree(objTree, "jasons-game/inscriptions/forged by", "forged1")
	require.Nil(t, err)

	for _, inscriptionKey := range []string{"type", "material", "age", "weight"} {
		objTree, err = net.UpdateChainTree(objTree, fmt.Sprintf("jasons-game/inscriptions/%s", inscriptionKey), fmt.Sprintf("%s1", inscriptionKey))
		require.Nil(t, err)
	}
	_, err = game.CreateObjectOnTree(net, "artifact-bad", objTree)
	require.Nil(t, err)

	err = playerInventory.ForceAdd(objTree.MustId())
	require.Nil(t, err)
	// end fake artifact

	go func() {
		<-ctx.Done()
		rootCtx.Stop(gamePid)
		rootCtx.Stop(simulatedUI)
	}()

	return stream, gamePid
}

func runCommands(t *testing.T, cmdsAndResponses []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, gamePid := setupTestArcadiaCourt(t, ctx)

	for i := 0; i < len(cmdsAndResponses); i = i + 2 {
		cmd := cmdsAndResponses[i]
		response := cmdsAndResponses[i+1]
		stream.ExpectMessage(response, 20*time.Second)
		actor.EmptyRootContext.Send(gamePid, &jasonsgame.UserInput{Message: cmd})
		stream.Wait()
	}
}

func TestArcadiaCourt(t *testing.T) {
	runCommands(t, []string{
		"go altar1", "altar1 description",
		"drop artifact-1", "object has been dropped",
		"go altar2", "altar2 description",
		"drop artifact-2", "object has been dropped",
		"go altar3", "altar3 description",
		"drop artifact-3", "object has been dropped",
		"go altar4", "altar4 description",
		"drop artifact-4", "object has been dropped",
		"go altar5", "altar5 description",
		"drop artifact-5", "object has been dropped",
		"go main", "main description",
		"pick up spawn-obj", "won arcadia",
		"look in bag", "test-arcadia-prize",
	})
}

func TestArcadiaCourtFailOrder(t *testing.T) {
	// switch altar 1 & 2 artifacts
	runCommands(t, []string{
		"go altar1", "altar1 description",
		"drop artifact-2", "object has been dropped",
		"go altar2", "altar2 description",
		"drop artifact-1", "object has been dropped",
		"go altar3", "altar3 description",
		"drop artifact-3", "object has been dropped",
		"go altar4", "altar4 description",
		"drop artifact-4", "object has been dropped",
		"go altar5", "altar5 description",
		"drop artifact-5", "object has been dropped",
		"go main", "main description",
		"pick up spawn-obj", "your altar placement is incorrect",
	})
}

func TestArcadiaCourtFailOwnership(t *testing.T) {
	// use player faked artifact-bad
	runCommands(t, []string{
		"go altar1", "altar1 description",
		"drop artifact-bad", "object has been dropped",
		"go altar2", "altar2 description",
		"drop artifact-2", "object has been dropped",
		"go altar3", "altar3 description",
		"drop artifact-3", "object has been dropped",
		"go altar4", "altar4 description",
		"drop artifact-4", "object has been dropped",
		"go altar5", "altar5 description",
		"drop artifact-5", "object has been dropped",
		"go main", "main description",
		"pick up spawn-obj", "your altar placement is incorrect",
	})
}
