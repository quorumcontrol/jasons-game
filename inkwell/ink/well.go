package ink

import (
	"context"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"

	"github.com/quorumcontrol/jasons-game/inkwell/inkwell"
	"github.com/quorumcontrol/jasons-game/network"
)

const InkwellChainTreeName = "inkwell"

var log = logging.Logger("ink")

type Well interface {
	TokenName() *consensus.TokenName
	RequestInk(amount uint64, destinationChainId string) (*transactions.TokenPayload, error)
	// only works in internal builds b/c Network.ReceiveInk just returns an error in public builds
	DepositInk(tokenPayload *transactions.TokenPayload) error
}

type ChainTreeInkwell struct {
	ct           *consensus.SignedChainTree
	net          network.Network
	tokenOwnerId string
}

type ChainTreeInkwellConfig struct {
	Net          network.Network
	ChainTreeDid string
}

var _ Well = &ChainTreeInkwell{}

func NewChainTreeInkwell(cfg ChainTreeInkwellConfig) (*ChainTreeInkwell, error) {
	ct, err := ensureChainTree(cfg.Net)

	if err != nil {
		return nil, err
	}

	fmt.Printf("INKWELL_DID=%s\n", ct.MustId())

	cti := &ChainTreeInkwell{
		ct:           ct,
		net:          cfg.Net,
		tokenOwnerId: cfg.ChainTreeDid,
	}

	return cti, nil
}

// ensureChainTree gets or creates a new inkwell chaintree.
// Note that this chaintree doesn't typically own the ink token; it just contains some
// that was sent to it.
func ensureChainTree(net network.Network) (*consensus.SignedChainTree, error) {
	existing, err := net.GetChainTreeByName(InkwellChainTreeName)
	if existing == nil {
		if err != nil {
			return nil, errors.Wrap(err, "error checking for existing inkwell chaintree")
		}
		return net.CreateNamedChainTree(InkwellChainTreeName)
	}

	return existing, nil
}

func (cti *ChainTreeInkwell) DepositInk(tokenPayload *transactions.TokenPayload) error {
    return cti.net.ReceiveInk(cti.ct, tokenPayload)
}

func (cti *ChainTreeInkwell) RequestInk(amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	tokenName := cti.TokenName()

	tokenLedger := consensus.NewTreeLedger(cti.ct.ChainTree.Dag, tokenName)

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
		return nil, fmt.Errorf("ink token balance %d is insufficient to fulfill request for %d", tokenBalance, amount)
	}

	return cti.net.SendInk(cti.ct, tokenName, amount, destinationChainId)
}

func (cti *ChainTreeInkwell) TokenName() *consensus.TokenName {
	return &consensus.TokenName{ChainTreeDID: cti.tokenOwnerId, LocalName: "ink"}
}

type InkActor struct {
	parentCtx context.Context
	group     *types.NotaryGroup
	dataStore datastore.Batching
	net       network.Network
	inkSource Well
	tokenName *consensus.TokenName
}

type InkActorConfig struct {
	Group     *types.NotaryGroup
	DataStore datastore.Batching
	Net       network.Network
	InkSource Well
	TokenName *consensus.TokenName
}

func NewInkActor(ctx context.Context, cfg InkActorConfig) *InkActor {
	return &InkActor{
		parentCtx: ctx,
		group:     cfg.Group,
		dataStore: cfg.DataStore,
		net:       cfg.Net,
		inkSource: cfg.InkSource,
		tokenName: cfg.TokenName,
	}
}

func (i *InkActor) Start(arCtx *actor.RootContext) {
	act := arCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return i
	}))

	go func() {
		<-i.parentCtx.Done()
		arCtx.Stop(act)
	}()
}

func (i *InkActor) Receive(aCtx actor.Context) {
	switch msg := aCtx.Message().(type) {
	case *actor.Started:
		log.Info("ink actor started")
	case *inkwell.InkRequest:
		log.Infof("ink actor received ink request: %+v", msg)
		tokenPayload, err := i.inkSource.RequestInk(msg.Amount, msg.DestinationChainId)
		if err != nil {
			aCtx.Respond(&inkwell.InkResponse{
				Error: err.Error(),
			})
			return
		}

		aCtx.Respond(&inkwell.InkResponse{
			Token: tokenPayload.String(),
		})
	}
}
