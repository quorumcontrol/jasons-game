package game

import (
	"context"

	"github.com/99designs/keyring"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipfs/go-datastore"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
)

const keyringPrivateKeyName = "private-key"

type AuthenticatedSessionConfig struct {
	UiActor     *actor.PID
	DataStore   datastore.Batching
	NotaryGroup *types.NotaryGroup
	Keyring     keyring.Keyring
}

type AuthenticatedSession struct {
	parentCtx context.Context
	ui        *actor.PID
	ds        datastore.Batching
	group     *types.NotaryGroup
	childPid  *actor.PID
	keyring   keyring.Keyring
}

func NewAuthenticatedSessionProps(ctx context.Context, cfg *AuthenticatedSessionConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &AuthenticatedSession{
			parentCtx: ctx,
			ds:        cfg.DataStore,
			ui:        cfg.UiActor,
			group:     cfg.NotaryGroup,
			keyring:   cfg.Keyring,
		}
	})
}

func (s *AuthenticatedSession) initialize(actorCtx actor.Context) {
	pkey, err := s.keyring.Get(keyringPrivateKeyName)

	if pkey.Data != nil && err == nil {
		key, err := crypto.ToECDSA(pkey.Data)
		if err != nil {
			panic(err)
		}

		net, err := network.NewRemoteNetworkWithConfig(s.parentCtx, &network.RemoteNetworkConfig{
			NotaryGroup:   s.group,
			KeyValueStore: s.ds,
			SigningKey:    key,
			NetworkKey:    key,
		})

		if err != nil {
			panic(err)
		}

		playerTree, err := net.FindOrCreatePassphraseTree("player")
		if err != nil {
			panic(err)
		}

		gameCfg := &GameConfig{
			PlayerTree: NewPlayerTree(net, playerTree),
			UiActor:    s.ui,
			Network:    net,
			DataStore:  s.ds,
		}

		s.childPid = actorCtx.Spawn(NewGameProps(gameCfg))
	} else {
		net, err := network.NewRemoteNetworkWithConfig(s.parentCtx, &network.RemoteNetworkConfig{
			NotaryGroup:   s.group,
			KeyValueStore: s.ds,
			SigningKey:    nil,
		})
		if err != nil {
			panic(err)
		}
		s.childPid = actorCtx.Spawn(NewLoginProps(&LoginConfig{
			UiActor: s.ui,
			Network: net,
			Keyring: s.keyring,
		}))
	}

	actorCtx.Send(s.ui, &ui.SetGame{Game: s.childPid})
}

func (s *AuthenticatedSession) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		log.Debug("AuthenticatedSession: starting actor")
		s.initialize(actorCtx)
	case *actor.Stopping:
		log.Debug("AuthenticatedSession: stopping actor")
	case *actor.Restart:
		log.Info("AuthenticatedSession: restarting actor")
	case *ping:
		actorCtx.Respond(true)
	case *actor.Terminated:
		if s.childPid == msg.Who {
			s.childPid = nil
		}
		s.initialize(actorCtx)
	default:
		if s.childPid != nil {
			actorCtx.Forward(s.childPid)
		} else {
			log.Warningf("AuthenticatedSession: received message %v with no game actor to forward to", msg)
		}
	}
}
