package invites

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/jasons-game/inkfaucet/ink"
	"github.com/quorumcontrol/jasons-game/inkfaucet/inkfaucet"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("invites")

type InvitesActor struct {
	parentCtx context.Context
	handler   *actor.PID
	inkActor  *ink.InkActor
	net       network.Network
}

type InvitesActorConfig struct {
	InkActor *ink.InkActor
	InkDID   string
	Net      network.Network
}

func NewInvitesActor(ctx context.Context, cfg InvitesActorConfig) *InvitesActor {
	return &InvitesActor{
		parentCtx: ctx,
		inkActor:  cfg.InkActor,
		net:       cfg.Net,
	}
}

func (i *InvitesActor) Start(arCtx *actor.RootContext) {
	act := arCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return i
	}))

	i.handler = act

	go func() {
		<-i.parentCtx.Done()
		arCtx.Stop(act)
	}()
}

func (i *InvitesActor) PID() *actor.PID {
	return i.handler
}

func (i *InvitesActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		log.Debug("invites actor started")
	case *inkfaucet.InviteRequest:
		log.Debug("invites actor received invite request")
		i.handleInviteRequest(actorCtx)
	case *inkfaucet.InviteSubmission:
		log.Debug("invites actor received invite submission")
		i.handleInviteSubmission(actorCtx, msg)
	default:
		log.Warningf("invites actor received unknown message type %T: %+v", msg, msg)
	}
}

func (i *InvitesActor) handleInviteSubmission(actorCtx actor.Context, msg *inkfaucet.InviteSubmission) {
	encodedKey := msg.Invite
	serializedKey := base58.Decode(encodedKey)

	log.Debug("decoded invite")

	key, err := crypto.ToECDSA(serializedKey)
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debug("converted invite to key")

	inviteChainTreeDID := consensus.EcdsaPubkeyToDid(key.PublicKey)

	log.Debugf("invite code converted to DID: %s", inviteChainTreeDID)

	inviteChainTree, err := i.net.GetTree(inviteChainTreeDID)
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debugf("got invite chaintree: %+v", inviteChainTree)

	if inviteChainTree == nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invite chaintree is nil",
		})
		return
	}

	playerChainTree, err := i.net.CreateLocalChainTree("player")
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debug("created new player chaintree")

	inviteTree, err := inviteChainTree.ChainTree.Tree(context.TODO())

	// TODO: Move all the ink transfer code to its own func
	tokenLedger := consensus.NewTreeLedger(inviteTree, i.net.InkTokenName())

	inkBalance, err := tokenLedger.Balance()
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debugf("depositing %d ink for player", inkBalance)

	err = i.net.DepositInk(inviteChainTree, key, inkBalance)
	if err != nil {
		log.Debugf("error depositing ink: %v", err)

		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debug("deposited  ink for player")

	err = i.net.DeleteTree(inviteChainTreeDID)
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debug("deleted invite chaintree")

	actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
		PlayerChainId: playerChainTree.MustId(),
	})
}
