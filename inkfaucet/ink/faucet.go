package ink

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("inkFaucet")

type Faucet interface {
	TokenName() *consensus.TokenName
	ChainTreeDID() string
	RequestInk(amount uint64, destinationChainId string) (*transactions.TokenPayload, error)
	// only works in internal builds b/c Network.ReceiveInk just returns an error in public builds
	DepositInk(tokenPayload *transactions.TokenPayload) error
}

type ChainTreeInkFaucet struct {
	ctDID      string
	net        network.Network
	inkOwnerId string
}

type ChainTreeInkFaucetConfig struct {
	Net         network.Network
	InkOwnerDID string
	PrivateKey  *ecdsa.PrivateKey
}

var _ Faucet = &ChainTreeInkFaucet{}

func NewChainTreeInkFaucet(cfg ChainTreeInkFaucetConfig) (*ChainTreeInkFaucet, error) {
	ct, err := ensureChainTree(cfg.Net, cfg.PrivateKey)

	if err != nil {
		return nil, err
	}

	log.Infof("INK_FAUCET_DID=%s", ct.MustId())

	cti := &ChainTreeInkFaucet{
		ctDID:      ct.MustId(),
		net:        cfg.Net,
		inkOwnerId: cfg.InkOwnerDID,
	}

	return cti, nil
}

// ensureChainTree gets or creates a new inkFaucet chaintree.
// Note that this chaintree doesn't typically own the ink token; it just contains some
// that was sent to it.
func ensureChainTree(net network.Network, key *ecdsa.PrivateKey) (*consensus.SignedChainTree, error) {
	did := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())

	existing, err := net.GetTree(did)
	if err != nil {
		return nil, errors.Wrap(err, "error checking for existing inkFaucet chaintree")
	}

	if existing == nil {
		ct, err := net.CreateChainTreeWithKey(key)
		if err != nil {
			return nil, err
		}

		err = net.TreeStore().SaveTreeMetadata(ct)
		if err != nil {
			return nil, err
		}

		return ct, nil
	}

	return existing, nil
}

func (cti *ChainTreeInkFaucet) chainTree() (*consensus.SignedChainTree, error) {
	var err error

	tip := cid.Undef

	rn, ok := cti.net.(*network.RemoteNetwork)
	if ok {
		tip, err = rn.Tupelo.GetTip(cti.ctDID)
		if err != nil {
			return nil, err
		}
	}

	if tip.Equals(cid.Undef) {
		ct, err := cti.net.GetTree(cti.ctDID)
		if err != nil {
			return nil, err
		}

		return ct, nil
	}

	ct, err := cti.net.GetTreeByTip(tip)
	if err != nil {
		return nil, err
	}

	return ct, nil
}

func (cti *ChainTreeInkFaucet) DepositInk(tokenPayload *transactions.TokenPayload) error {
	return cti.net.ReceiveInk(tokenPayload)
}

func (cti *ChainTreeInkFaucet) RequestInk(amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	tokenName := cti.TokenName()

	if !tokenName.IsCanonical() {
		return nil, errors.Errorf("token name %s is not canonical", tokenName)
	}

	log.Debugf("ink request canonical token name: %s", cti.TokenName())

	ct, err := cti.chainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error requesting ink")
	}

	log.Debugf("ink faucet chaintree: %s", ct.ChainTree.Dag.Dump(context.TODO()))

	inkfaucetTree, err := ct.ChainTree.Tree(context.TODO())
	if err != nil {
		return nil, errors.Wrap(err, "error getting inkFaucet tree for ink request")
	}

	tokenLedger := consensus.NewTreeLedger(inkfaucetTree, tokenName)

	tokenExists, err := tokenLedger.TokenExists()
	if err != nil {
		return nil, errors.Wrap(err, "error checking for ink token existence")
	}

	log.Debugf("ink faucet token exists? %v", tokenExists)

	if !tokenExists {
		return nil, errors.Wrapf(err, "ink token %s does not exist", tokenName)
	}

	tokenBalance, err := tokenLedger.Balance()
	if err != nil {
		return nil, errors.Wrap(err, "error getting ink token balance")
	}

	log.Debugf("ink faucet token balance: %d", tokenBalance)

	if tokenBalance < amount {
		return nil, errors.Errorf("ink token balance %d is insufficient to fulfill request for %d", tokenBalance, amount)
	}

	return cti.net.SendInk(amount, destinationChainId)
}

func (cti *ChainTreeInkFaucet) TokenName() *consensus.TokenName {
	return cti.net.InkTokenName()
}

func (cti *ChainTreeInkFaucet) ChainTreeDID() string {
	return cti.ctDID
}

type InkActor struct {
	parentCtx context.Context
	inkFaucet Faucet
	handler   *actor.PID
}

type InkActorConfig struct {
	InkFaucet Faucet
}

func NewInkActor(ctx context.Context, cfg InkActorConfig) *InkActor {
	return &InkActor{
		parentCtx: ctx,
		inkFaucet: cfg.InkFaucet,
	}
}

func (i *InkActor) Start(arCtx *actor.RootContext) {
	act := arCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return i
	}))

	i.handler = act

	go func() {
		<-i.parentCtx.Done()
		arCtx.Stop(act)
	}()
}

func (i *InkActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		log.Info("ink actor started")
	case *inkfaucet.InkRequest:
		log.Infof("ink actor received ink request: %+v", msg)
		tokenPayload, err := i.inkFaucet.RequestInk(msg.Amount, msg.DestinationChainId)
		if err != nil {
			msg := fmt.Sprintf("error requesting ink: %v", err)
			log.Error(msg)
			actorCtx.Respond(&inkfaucet.InkResponse{
				Error: msg,
			})
			return
		}

		log.Debugf("ink actor got token payload: %+v", tokenPayload)

		var response *inkfaucet.InkResponse

		serializedTokenPayload, err := proto.Marshal(tokenPayload)
		if err != nil {
			msg := fmt.Sprintf("error marshalling token payload: %v", err)
			log.Error(msg)
			response = &inkfaucet.InkResponse{
				Error: msg,
			}
			actorCtx.Respond(response)
			return
		}

		log.Debug("ink actor serialized token payload")

		response = &inkfaucet.InkResponse{
			Token: serializedTokenPayload,
		}

		log.Debugf("ink actor sending response: %+v", response)

		actorCtx.Respond(response)
	default:
		log.Warningf("ink actor received unknown message type %T", msg)
	}
}

func (i *InkActor) PID() *actor.PID {
	return i.handler
}
