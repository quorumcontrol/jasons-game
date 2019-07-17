package ink

import (
	"context"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/gogo/protobuf/proto"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"

	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
	"github.com/quorumcontrol/jasons-game/network"
)

const InkFaucetChainTreeName = "inkFaucet"

var log = logging.Logger("ink")

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
}

var _ Faucet = &ChainTreeInkFaucet{}

func NewChainTreeInkFaucet(cfg ChainTreeInkFaucetConfig) (*ChainTreeInkFaucet, error) {
	ct, err := ensureChainTree(cfg.Net)

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
func ensureChainTree(net network.Network) (*consensus.SignedChainTree, error) {
	existing, err := net.GetChainTreeByName(InkFaucetChainTreeName)
	if err != nil {
		return nil, errors.Wrap(err, "error checking for existing inkFaucet chaintree")
	}

	if existing == nil {
		return net.CreateNamedChainTree(InkFaucetChainTreeName)
	}

	return existing, nil
}

func (cti *ChainTreeInkFaucet) chainTree() (*consensus.SignedChainTree, error) {
	ct, err := cti.net.GetTree(cti.ctDID)
	if err != nil {
		return nil, err
	}

	return ct, nil
}

func (cti *ChainTreeInkFaucet) DepositInk(tokenPayload *transactions.TokenPayload) error {
	ct, err := cti.chainTree()
	if err != nil {
		return errors.Wrap(err, "error depositing ink")
	}

	return cti.net.ReceiveInk(ct, tokenPayload)
}

func (cti *ChainTreeInkFaucet) RequestInk(amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	tokenName := cti.TokenName()

	ct, err := cti.chainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error requesting ink")
	}

	inkfaucetTree, err := ct.ChainTree.Tree(context.TODO())
	if err != nil {
		return nil, errors.Wrap(err, "error getting inkFaucet tree for ink request")
	}

	tokenLedger := consensus.NewTreeLedger(inkfaucetTree, tokenName)

	tokenExists, err := tokenLedger.TokenExists()
	if err != nil {
		return nil, errors.Wrap(err, "error checking for ink token existence")
	}

	if !tokenExists {
		return nil, errors.Wrapf(err, "ink token %s does not exist", tokenName)
	}

	tokenBalance, err := tokenLedger.Balance()
	if err != nil {
		return nil, errors.Wrap(err, "error getting ink token balance")
	}

	if tokenBalance < amount {
		return nil, errors.Wrapf(err, "ink token balance %d is insufficient to fulfill request for %d", tokenBalance, amount)
	}

	return cti.net.SendInk(ct, tokenName, amount, destinationChainId)
}

func (cti *ChainTreeInkFaucet) TokenName() *consensus.TokenName {
	return &consensus.TokenName{ChainTreeDID: cti.inkOwnerId, LocalName: "ink"}
}

func (cti *ChainTreeInkFaucet) ChainTreeDID() string {
	return cti.ctDID
}

type InkActor struct {
	parentCtx context.Context
	group     *types.NotaryGroup
	dataStore datastore.Batching
	net       network.Network
	inkFaucet Faucet
	tokenName *consensus.TokenName
	handler   *actor.PID
}

type InkActorConfig struct {
	Group     *types.NotaryGroup
	DataStore datastore.Batching
	Net       network.Network
	InkFaucet Faucet
	TokenName *consensus.TokenName
}

func NewInkActor(ctx context.Context, cfg InkActorConfig) *InkActor {
	return &InkActor{
		parentCtx: ctx,
		group:     cfg.Group,
		dataStore: cfg.DataStore,
		net:       cfg.Net,
		inkFaucet: cfg.InkFaucet,
		tokenName: cfg.TokenName,
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
			actorCtx.Respond(&inkfaucet.InkResponse{
				Error: err.Error(),
			})
			return
		}

		var response *inkfaucet.InkResponse

		serializedTokenPayload, err := proto.Marshal(tokenPayload)
		if err != nil {
			response := &inkfaucet.InkResponse{
				Error: fmt.Sprintf("error serializing dev ink token payload: %v", err),
			}

			actorCtx.Respond(response)
			return
		}

		response = &inkfaucet.InkResponse{
			Token: serializedTokenPayload,
		}

		actorCtx.Respond(response)
	}
}

func (i *InkActor) PID() *actor.PID {
	return i.handler
}
