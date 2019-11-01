package game

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/99designs/keyring"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/game/signup"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/tyler-smith/go-bip39"
)

const loginCmdSignUp = "sign up"
const loginCmdRecover = "recover"
const loginCmdRecoveryPhrase = "recovery phrase"

var loginWelcomeMessage = fmt.Sprintf("Please type `%s` or `%s` followed by your email to continue.", loginCmdSignUp, loginCmdRecover)
var loginRecoveryMessage = fmt.Sprintf("Please type `%s` followed by the recovery phrase that was provided when you signed up.", loginCmdRecoveryPhrase)

const loginRecoverySuccessMessage = "Account recovery successful! Transporting you to the land of the fae to continue your adventure."
const loginRecoveryFailureMessage = "No account exists for that email and recovery phrase."
const loginEmailErrorMessage = "You must provide a valid email after `%s`."
const loginProvideSeedMessage = `Signing up with %s.

Below is your recovery phrase. Please write this down in a safe place.

%s

Once you have written down your recovery phrase, type "portal to fae" to begin your adventure.
`

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type LoginConfig struct {
	UiActor *actor.PID
	Keyring keyring.Keyring
	Network network.Network
}

type Login struct {
	ui      *actor.PID
	net     network.Network
	cmds    []string
	keyring keyring.Keyring
	state   map[string]string
}

func NewLoginProps(cfg *LoginConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &Login{
			ui:      cfg.UiActor,
			net:     cfg.Network,
			keyring: cfg.Keyring,
			cmds:    []string{},
			state:   make(map[string]string),
		}
	})
}

func (l *Login) commandUpdate(actorCtx actor.Context) {
	cmdUpdate := &jasonsgame.CommandUpdate{Commands: append(l.cmds, "help")}
	actorCtx.Send(l.ui, cmdUpdate)
}

func (l *Login) setDefaultCommands(actorCtx actor.Context) {
	l.cmds = []string{loginCmdSignUp, loginCmdRecover}
	l.commandUpdate(actorCtx)
}

func (l *Login) sendUserMessage(actorCtx actor.Context, mesgInter interface{}) {
	if sender := actorCtx.Sender(); sender != nil {
		actorCtx.Respond(&jasonsgame.CommandReceived{})
	}
	actorCtx.Send(l.ui, formatUserMessage(mesgInter))
}

func (l *Login) Receive(actorCtx actor.Context) {
	keyringKey, err := l.keyring.Get(keyringPrivateKeyName)
	// Key exists, no need to login
	if keyringKey.Data != nil && err == nil {
		actorCtx.Stop(actorCtx.Self())
		return
	}

	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		l.setDefaultCommands(actorCtx)
		l.sendUserMessage(actorCtx, loginWelcomeMessage)
	case *actor.Stopping:
		log.Info("login actor stopping")
	case *jasonsgame.UserInput:
		m := msg.Message

		switch true {
		case strings.HasPrefix(m, loginCmdSignUp):
			signupClient, err := signup.NewClient(l.net)
			if err != nil {
				log.Error(errors.Wrap(err, "error fetching signup client"))
				l.sendUserMessage(actorCtx, "jasonsgame is currently unavailable, please try again later")
				return
			}

			email := strings.TrimSpace(strings.TrimPrefix(m, loginCmdSignUp))

			if len(email) == 0 || !emailRegex.MatchString(email) {
				l.sendUserMessage(actorCtx, fmt.Sprintf(loginEmailErrorMessage, loginCmdSignUp))
				return
			}

			entropy, err := bip39.NewEntropy(256)
			if err != nil {
				panic(errors.Wrap(err, "error creating seed"))
			}
			mnemonic, err := bip39.NewMnemonic(entropy)
			if err != nil {
				panic(errors.Wrap(err, "error creating mnemonic"))
			}
			key, err := consensus.PassPhraseKey([]byte(mnemonic), []byte(email))
			if err != nil {
				panic(errors.Wrap(err, "error generating key"))
			}
			err = signupClient.Signup(email, consensus.EcdsaPubkeyToDid(key.PublicKey))
			if err != nil {
				panic(errors.Wrap(err, "error signing up"))
			}

			mnemonicWrapped := strings.Join(strings.Split(mnemonic, " ")[0:12], " ") + "\n" + strings.Join(strings.Split(mnemonic, " ")[12:], " ")

			err = l.keyring.Set(keyring.Item{
				Key:   keyringPrivateKeyName,
				Label: "jasons-game",
				Data:  crypto.FromECDSA(key),
			})
			if err != nil {
				panic(errors.Wrap(err, "error saving key"))
			}

			l.sendUserMessage(actorCtx, fmt.Sprintf(loginProvideSeedMessage, email, mnemonicWrapped))
			l.cmds = []string{"portal to fae"}
		case strings.HasPrefix(m, "portal to fae"):
			actorCtx.Stop(actorCtx.Self())
			return
		case strings.HasPrefix(m, loginCmdRecoveryPhrase):
			mnemonic := strings.TrimSpace(strings.TrimPrefix(m, loginCmdRecoveryPhrase))

			email := l.state["email"]
			delete(l.state, "email")

			key, err := consensus.PassPhraseKey([]byte(mnemonic), []byte(email))
			if err != nil {
				panic(errors.Wrap(err, "error generating key"))
			}
			keyBytes := crypto.FromECDSA(key)

			// Check player tree exists
			seed := sha256.Sum256([]byte("player"))
			playerTreeKey, err := consensus.PassPhraseKey(keyBytes, seed[:32])
			if err != nil {
				panic(errors.Wrap(err, "error checking player tree"))
			}

			playerTree, err := l.net.GetTree(consensus.EcdsaPubkeyToDid(playerTreeKey.PublicKey))
			if err != nil {
				panic(errors.Wrap(err, "error checking player tree"))
			}

			if playerTree == nil {
				l.setDefaultCommands(actorCtx)
				l.sendUserMessage(actorCtx, loginRecoveryFailureMessage)
				return
			}

			err = l.keyring.Set(keyring.Item{
				Key:   keyringPrivateKeyName,
				Label: "jasons-game",
				Data:  keyBytes,
			})
			if err != nil {
				panic(errors.Wrap(err, "error saving key"))
			}

			l.sendUserMessage(actorCtx, loginRecoverySuccessMessage)
			actorCtx.Stop(actorCtx.Self())
			return
		case strings.HasPrefix(m, loginCmdRecover):
			email := strings.TrimSpace(strings.TrimPrefix(m, loginCmdRecover))
			if len(email) == 0 || !emailRegex.MatchString(email) {
				l.sendUserMessage(actorCtx, fmt.Sprintf(loginEmailErrorMessage, loginCmdRecover))
				return
			}

			l.state["email"] = email
			l.sendUserMessage(actorCtx, loginRecoveryMessage)
			l.cmds = []string{loginCmdSignUp, loginCmdRecoveryPhrase}
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
