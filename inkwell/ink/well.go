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

	"github.com/quorumcontrol/jasons-game/inkwell/inkwell"
	"github.com/quorumcontrol/jasons-game/network"
)

const InkwellChainTreeName = "inkwell"

var log = logging.Logger("ink")

type Well interface {
	TokenName() *consensus.TokenName
	ChainTreeDID() string
	RequestInk(amount uint64, destinationChainId string) (*transactions.TokenPayload, error)
	// only works in internal builds b/c Network.ReceiveInk just returns an error in public builds
	DepositInk(tokenPayload *transactions.TokenPayload) error
}

type ChainTreeInkwell struct {
	ctDID      string
	net        network.Network
	inkOwnerId string
}

type ChainTreeInkwellConfig struct {
	Net         network.Network
	InkOwnerDID string
}

var _ Well = &ChainTreeInkwell{}

func NewChainTreeInkwell(cfg ChainTreeInkwellConfig) (*ChainTreeInkwell, error) {
	ct, err := ensureChainTree(cfg.Net)

	if err != nil {
		return nil, err
	}

	fmt.Printf("INKWELL_DID=%s\n", ct.MustId())

	cti := &ChainTreeInkwell{
		ctDID:      ct.MustId(),
		net:        cfg.Net,
		inkOwnerId: cfg.InkOwnerDID,
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

func (cti *ChainTreeInkwell) chainTree() (*consensus.SignedChainTree, error) {
	ct, err := cti.net.GetTree(cti.ctDID)
	if err != nil {
		return nil, err
	}

	return ct, nil
}

func (cti *ChainTreeInkwell) DepositInk(tokenPayload *transactions.TokenPayload) error {
	ct, err := cti.chainTree()
	if err != nil {
		return errors.Wrap(err, "error depositing ink")
	}

    return cti.net.ReceiveInk(ct, tokenPayload)
}

func (cti *ChainTreeInkwell) RequestInk(amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	tokenName := cti.TokenName()

	fmt.Println("tokenName:", tokenName)

	ct, err := cti.chainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error requesting ink")
	}

	inkwellTree, err := ct.ChainTree.Tree(context.TODO())
	if err != nil {
		return nil, errors.Wrap(err, "error getting inkwell tree for ink request")
	}

	fmt.Println("inkwell tree:", inkwellTree.Dump(context.TODO()))

	tokenLedger := consensus.NewTreeLedger(inkwellTree, tokenName)

	tokenExists, err := tokenLedger.TokenExists()
	if err != nil {
		return nil, errors.Wrap(err, "error checking for ink token existence")
	}
	fmt.Printf("tokenExists: %v\n", tokenExists)

	if !tokenExists {
		return nil, errors.Wrapf(err, "ink token %s does not exist", tokenName)
	}

	fmt.Println("token exists")

	tokenBalance, err := tokenLedger.Balance()
	if err != nil {
		return nil, errors.Wrap(err, "error getting ink token balance")
	}

	fmt.Println("token balance:", tokenBalance)

	if tokenBalance < amount {
		return nil, errors.Wrapf(err, "ink token balance %d is insufficient to fulfill request for %d", tokenBalance, amount)
	}

	fmt.Println("About to call net.SendInk")

	return cti.net.SendInk(ct, tokenName, amount, destinationChainId)
}

func (cti *ChainTreeInkwell) TokenName() *consensus.TokenName {
	return &consensus.TokenName{ChainTreeDID: cti.inkOwnerId, LocalName: "ink"}
}

func (cti *ChainTreeInkwell) ChainTreeDID() string {
	return cti.ctDID
}

type InkActor struct {
	parentCtx context.Context
	group     *types.NotaryGroup
	dataStore datastore.Batching
	net       network.Network
	inkwell   Well
	tokenName *consensus.TokenName
	handler   *actor.PID
}

type InkActorConfig struct {
	Group     *types.NotaryGroup
	DataStore datastore.Batching
	Net       network.Network
	Inkwell   Well
	TokenName *consensus.TokenName
}

func NewInkActor(ctx context.Context, cfg InkActorConfig) *InkActor {
	return &InkActor{
		parentCtx: ctx,
		group:     cfg.Group,
		dataStore: cfg.DataStore,
		net:       cfg.Net,
		inkwell:   cfg.Inkwell,
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
	case *inkwell.InkRequest:
		log.Infof("ink actor received ink request: %+v", msg)
		tokenPayload, err := i.inkwell.RequestInk(msg.Amount, msg.DestinationChainId)
		if err != nil {
			actorCtx.Respond(&inkwell.InkResponse{
				Error: err.Error(),
			})
			return
		}

		var response *inkwell.InkResponse

		serializedTokenPayload, err := proto.Marshal(tokenPayload)
		if err != nil {
			response := &inkwell.InkResponse{
				Error: fmt.Sprintf("error serializing dev ink token payload: %v", err),
			}

			actorCtx.Respond(response)
			return
		}

		response = &inkwell.InkResponse{
			Token: serializedTokenPayload,
		}

		actorCtx.Respond(response)
	}
}

func (i *InkActor) PID() *actor.PID {
	return i.handler
}
