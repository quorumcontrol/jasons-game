package signup

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

var SignupsPath = []string{"tree", "data", "track"}

type ServerActor struct {
	middleware.LogAwareHolder

	network       network.Network
	encryptionKey *ecdsa.PrivateKey
	tree          *consensus.SignedChainTree
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
	if len(msg.Encrypted) == 0 {
		return
	}

	eciesKey := ecies.ImportECDSA(s.encryptionKey)

	transitDecrypted, err := eciesKey.Decrypt(msg.Encrypted, nil, nil)
	if err != nil {
		s.Log.Error(errors.Wrap(err, "error decrypting"))
		return
	}

	signup := &jasonsgame.SignupMessage{}
	err = proto.Unmarshal(transitDecrypted, signup)
	if err != nil {
		s.Log.Error(errors.Wrap(err, "error unmarshaling decrypted payload"))
		return
	}

	s.Log.Infof("signup received for %s with %s", signup.GetDid(), signup.GetEmail())

	storageEncrypted, err := ecies.Encrypt(rand.Reader, ecies.ImportECDSAPublic(s.network.PublicKey()), transitDecrypted, nil, nil)
	if err != nil {
		s.Log.Error(errors.Wrap(err, "error encrypting for storage"))
		return
	}

	idSha := sha256.Sum256(storageEncrypted)
	id := hex.EncodeToString(idSha[:32])
	s.tree, err = s.network.UpdateChainTree(s.tree, strings.Join(append(SignupsPath[2:], id[0:1], id[1:2], id[2:3], id, "z"), "/"), storageEncrypted)
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

	encryptionKeySeed := sha256.Sum256([]byte("signups-encyrption"))
	s.encryptionKey, err = consensus.PassPhraseKey(crypto.FromECDSA(s.network.PrivateKey()), encryptionKeySeed[:32])
	if err != nil {
		panic(errors.Wrap(err, "setting up passphrase encryption keys"))
	}

	if storedPubKey, _, _ := s.tree.ChainTree.Dag.Resolve(context.Background(), encryptionPubKeyPath); storedPubKey == nil {
		s.tree, err = s.network.UpdateChainTree(s.tree, strings.Join(encryptionPubKeyPath[2:], "/"), crypto.FromECDSAPub(&s.encryptionKey.PublicKey))

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
