package signup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

const encryptionPubKey = "0x04ff1f17da517684f232afbae0a347ed459f339a636de30ae51c1cd49c71968d9e966ad4ffdff949ef1728f016b1129e372b139eaafdf88232df13fb64e8b38d66"

type ServerActor struct {
	middleware.LogAwareHolder
	network network.Network
	tree    *consensus.SignedChainTree
}

func NewServerProps(net network.Network) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &ServerActor{
			network: net,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (s *ServerActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		s.initialize(actorCtx)
	case *jasonsgame.SignupMessageEncrypted:
		s.handleSignup(actorCtx, msg)
	default:
		s.Log.Errorf("Unhandled message %v", msg)
	}
}

func (s *ServerActor) handleSignup(actorCtx actor.Context, msg *jasonsgame.SignupMessageEncrypted) {
	var err error
	if len(msg.Encrypted) == 0 {
		return
	}
	idSha := sha256.Sum256(msg.Encrypted)
	id := hex.EncodeToString(idSha[:32])
	s.tree, err = s.network.UpdateChainTree(s.tree, strings.Join([]string{"track", id[0:1], id[1:2], id[2:3], id, "z"}, "/"), msg.Encrypted)
	if err != nil {
		s.Log.Error(errors.Wrap(err, "error updating chain tree"))
		panic("error updating chain tree")
	}
}

func (s *ServerActor) initialize(actorCtx actor.Context) {
	var err error

	s.tree, err = s.network.FindOrCreatePassphraseTree("signups")
	if err != nil {
		panic(errors.Wrap(err, "getting passphrase chaintree"))
	}

	fmt.Printf("Signup service started with %s\n", s.tree.MustId())

	ecdsaPubkeyBytes, err := hexutil.Decode(encryptionPubKey)
	if err != nil {
		panic(fmt.Sprintf("error hexutil decoding storage encryption key: %v", err))
	}

	// This is just to ensure its valid key, but bytes on decode and FromECDSAPub should be the same
	ecdsaPubKey, err := crypto.UnmarshalPubkey(ecdsaPubkeyBytes)
	if err != nil {
		panic(fmt.Sprintf("error unmarshalling storage encryption key: %v", err))
	}

	if storedPubKey, _, _ := s.tree.ChainTree.Dag.Resolve(context.Background(), encryptionPubKeyPath); storedPubKey == nil {
		s.tree, err = s.network.UpdateChainTree(s.tree, strings.Join(encryptionPubKeyPath[2:], "/"), crypto.FromECDSAPub(ecdsaPubKey))

		if err != nil {
			panic(errors.Wrap(err, "setting encryption key"))
		}
	}

	client, err := NewClient(s.network)
	if err != nil {
		panic(err)
	}

	if client.Did() != s.tree.MustId() {
		panic(fmt.Sprintf("mismatched dids for client/server on signups: loaded %s, expected %s", s.tree.MustId(), client.Did()))
	}

	actorCtx.Spawn(s.network.Community().NewSubscriberProps(client.Topic()))
}
