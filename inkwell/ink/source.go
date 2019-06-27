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

var log = logging.Logger("ink")

type Source interface {
	TokenName() *consensus.TokenName
	DepositInk(tokenPayload *transactions.TokenPayload) error
	RequestInk(amount uint64, destinationChainId string) (*transactions.TokenPayload, error)
}

type ChainTreeInkSource struct {
	ct           *consensus.SignedChainTree
	net          network.Network
	tokenOwnerId string
}

type ChainTreeInkSourceConfig struct {
	Net          network.Network
	ChainTreeDid string
}

var _ Source = &ChainTreeInkSource{}

func NewChainTreeInkSource(cfg ChainTreeInkSourceConfig) (*ChainTreeInkSource, error) {
	ct, err := ensureChainTree(cfg.Net)

	if err != nil {
		return nil, err
	}

	fmt.Printf("INK_SOURCE_DID=%s\n", ct.MustId())

	ctis := &ChainTreeInkSource{
		ct:           ct,
		net:          cfg.Net,
		tokenOwnerId: cfg.ChainTreeDid,
	}

	return ctis, nil
}

// ensureChainTree gets or creates a new ink-source chaintree.
// Note that this chaintree doesn't typically own the ink token; it just contains some
// that was sent to it.
func ensureChainTree(net network.Network) (*consensus.SignedChainTree, error) {
	existing, err := net.GetChainTreeByName("ink-source")
	if existing == nil {
		if err != nil {
			return nil, errors.Wrap(err, "error checking for existing ink-source chaintree")
		}
		return net.CreateNamedChainTree("ink-source")
	}

	return existing, nil
}

func (ctis *ChainTreeInkSource) DepositInk(tokenPayload *transactions.TokenPayload) error {
    return ctis.net.ReceiveInk(ctis.ct, tokenPayload)
}

func (ctis *ChainTreeInkSource) RequestInk(amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	tokenName := ctis.TokenName()

	tokenLedger := consensus.NewTreeLedger(ctis.ct.ChainTree.Dag, tokenName)

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

	return ctis.net.SendInk(ctis.ct, tokenName, amount, destinationChainId)
}

func (ctis *ChainTreeInkSource) TokenName() *consensus.TokenName {
	return &consensus.TokenName{ChainTreeDID: ctis.tokenOwnerId, LocalName: "ink"}
}

type InkActor struct {
	parentCtx context.Context
	group     *types.NotaryGroup
	dataStore datastore.Batching
	net       network.Network
	inkSource Source
	tokenName *consensus.TokenName
}

type InkActorConfig struct {
	Group     *types.NotaryGroup
	DataStore datastore.Batching
	Net       network.Network
	InkSource Source
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
