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
	inkDID    string
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
		inkDID:    cfg.InkDID,
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

	key, err := crypto.ToECDSA(serializedKey)
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	inviteChainTree, err := i.net.GetChainTreeByName(encodedKey)
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	if inviteChainTree == nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	playerChainTree, err := i.net.CreateNamedChainTree("player")
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	playerChainTreeOwners, err := playerChainTree.Authentications()
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	newInviteChainTree, err := i.net.ChangeEphemeralChainTreeOwner(inviteChainTree, key, playerChainTreeOwners)
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	newInviteTree, err := newInviteChainTree.ChainTree.Tree(context.TODO())
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	inkTokenName := &consensus.TokenName{ChainTreeDID: i.inkDID, LocalName: "ink"}

	// TODO: Move all the ink transfer code to its own func
	tokenLedger := consensus.NewTreeLedger(newInviteTree, inkTokenName)

	inkBalance, err := tokenLedger.Balance()
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	tokenPayload, err := i.net.SendInk(newInviteChainTree, inkTokenName, inkBalance, playerChainTree.MustId())
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	err = i.net.ReceiveInk(playerChainTree, tokenPayload)
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	err = i.net.DeleteChainTreeByName(encodedKey)
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: "invalid invite code",
		})
		return
	}

	actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
		PlayerChainId: playerChainTree.MustId(),
	})
}

