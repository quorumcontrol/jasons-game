package game

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"
	"strings"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipfs/go-datastore"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/ui"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/scrypt"
)

var keystoreKey = datastore.NewKey("keystore-private-key")

type Login struct {
	game *Game
	cmds []string
	key  *ecdsa.PrivateKey
}

func NewLogin(game *Game) *Login {
	return &Login{game: game, cmds: []string{}}
}

func (l *Login) HasState() bool {
	hasState, err := l.game.ds.Has(keystoreKey)
	if err != nil {
		panic(err)
	}
	return hasState
}

func (l *Login) NewActorProps() *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return l
	})
}

func (l *Login) commandUpdate(actorCtx actor.Context) {
	cmdUpdate := &jasonsgame.CommandUpdate{Commands: append(l.cmds, "help")}
	actorCtx.Send(l.game.ui, cmdUpdate)
}

func (l *Login) sendUserMessage(actorCtx actor.Context, mesgInterface interface{}) {
	l.game.sendUserMessage(actorCtx, mesgInterface)
}

func (l *Login) encryptPrivateKey(key *ecdsa.PrivateKey, password []byte) ([]byte, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}

	cipherKey, err := scrypt.Key(password, nonce[:], 32768, 8, 1, 32)
	if err != nil {
		return nil, errors.Wrap(err, "error generating secret key")
	}
	var cipherKey32 [32]byte
	copy(cipherKey32[:], cipherKey)

	return secretbox.Seal(nonce[:], crypto.FromECDSA(key), &nonce, &cipherKey32), nil
}

func (l *Login) decryptPrivateKey(encryptedBytes []byte, password []byte) (*ecdsa.PrivateKey, error) {
	var nonce [24]byte
	copy(nonce[:], encryptedBytes[:cipherNonceLength])

	cipherKey, err := scrypt.Key(password, nonce[:], 32768, 8, 1, 32)
	if err != nil {
		return nil, errors.Wrap(err, "error generating secret key")
	}
	var cipherKey32 [32]byte
	copy(cipherKey32[:], cipherKey)

	unencryptedBytes, unsealSuccess := secretbox.Open(nil, encryptedBytes[cipherNonceLength:], &nonce, &cipherKey32)
	if !unsealSuccess {
		return nil, fmt.Errorf("Incorrect password")
	}

	return crypto.ToECDSA(unencryptedBytes)
}

func (l *Login) Receive(actorCtx actor.Context) {
	log.Debugf("login received: %+v", actorCtx.Message())

	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		actorCtx.Send(l.game.ui, &ui.SetGame{Game: actorCtx.Self()})

		var prompt string
		if l.HasState() {
			prompt = "Welcome back to Jason's Game! Please type either `login` followed by your password or `recover` followed by your email."
			l.cmds = []string{"login", "recover"}
		} else {
			prompt = "Welcome to Jason's Game! Please type `sign up` or `recover` followed by your email to continue."
			l.cmds = []string{"sign up", "recover"}
		}
		l.commandUpdate(actorCtx)
		l.sendUserMessage(actorCtx, prompt)
	case *actor.Stopping:
		log.Info("login actor stopping")
	case *jasonsgame.UserInput:
		l.game.acknowledgeReceipt(actorCtx)

		m := msg.Message

		switch true {
		case strings.HasPrefix(m, "sign up"):
			if l.HasState() {
				panic("restart, has state")
			}

			email := strings.TrimSpace(strings.TrimPrefix(m, "sign up"))

			entropy, err := bip39.NewEntropy(256)
			if err != nil {
				panic(errors.Wrap(err, "error creating seed"))
			}
			mnemonic, err := bip39.NewMnemonic(entropy)
			if err != nil {
				panic(errors.Wrap(err, "error creating mnemonic"))
			}
			l.key, err = consensus.PassPhraseKey([]byte(mnemonic), []byte(email))
			if err != nil {
				panic(errors.Wrap(err, "error generating key"))
			}

			mnemonicWrapped := strings.Join(strings.Split(mnemonic, " ")[0:12], " ") + "\n" + strings.Join(strings.Split(mnemonic, " ")[12:], " ")

			l.sendUserMessage(actorCtx, fmt.Sprintf(
				`Signing up with %s

Below is your recovery phrase in case you forget your password. Please write this down in a safe place.

%s

Now create your password with %s`, email, mnemonicWrapped, "`set-password`"))

			l.cmds = []string{"set-password"}
		case strings.HasPrefix(m, "login"):
			pw := strings.TrimSpace(strings.TrimPrefix(m, "login"))

			encyptedBytes, err := l.game.ds.Get(keystoreKey)
			if err != nil {
				panic(errors.Wrap(err, "not found"))
			}

			key, err := l.decryptPrivateKey(encyptedBytes, []byte(pw))
			if err != nil {
				panic(errors.Wrap(err, "bad login"))
			}

			l.game.network.(*network.RemoteNetwork).SetPrivateKey(key)
			playerTree, err := l.game.network.FindOrCreatePassphraseTree("player")
			if err != nil {
				panic(errors.Wrap(err, "error creating player tree"))
			}
			l.game.playerTree = NewPlayerTree(l.game.network, playerTree)
			l.game.behavior.Become(l.game.ReceiveGame)
			l.game.initializeGame(actorCtx)
			return
		case strings.HasPrefix(m, "set-password"):
			if l.key == nil {
				panic("restart, no key")
			}

			pw := strings.TrimSpace(strings.TrimPrefix(m, "set-password"))

			encryptedKey, err := l.encryptPrivateKey(l.key, []byte(pw))
			if err != nil {
				panic(errors.Wrap(err, "error encrypting keystore"))
			}

			err = l.game.ds.Put(keystoreKey, encryptedKey)
			if err != nil {
				panic(errors.Wrap(err, "error saving password"))
			}

			l.game.network.(*network.RemoteNetwork).SetPrivateKey(l.key)
			l.key = nil
			playerTree, err := l.game.network.FindOrCreatePassphraseTree("player")
			if err != nil {
				panic(errors.Wrap(err, "error creating player tree"))
			}
			l.game.playerTree = NewPlayerTree(l.game.network, playerTree)
			l.game.behavior.Become(l.game.ReceiveGame)
			l.game.initializeGame(actorCtx)
			return
		case strings.HasPrefix(m, "recover"):
			l.sendUserMessage(actorCtx, fmt.Sprintf("recover command received %s", msg.Message))
		case strings.HasPrefix(m, "help"):
			l.sendUserMessage(actorCtx, append(indentedList{"available commands:"}, l.cmds...))
		default:
			log.Debugf("login actor received unknown msg: %T %s", msg, msg.Message)
		}

		l.commandUpdate(actorCtx)
	case *jasonsgame.CommandUpdate:
		l.commandUpdate(actorCtx)
	case *ping:
		actorCtx.Respond(true)
	case *actor.Terminated:
		log.Info("login actor terminated")
	default:
		log.Debugf("login received: %v", msg)
	}
}
