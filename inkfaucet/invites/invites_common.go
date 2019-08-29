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

	playerChainTreeOwners, err := playerChainTree.Authentications()
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debug("get player chaintree owner")

	newInviteChainTree, err := i.net.ChangeChainTreeOwnerWithKey(inviteChainTree, key, playerChainTreeOwners)
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debug("changed owner of invite chaintree to player")

	newInviteTree, err := newInviteChainTree.ChainTree.Tree(context.TODO())
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	inkTokenName := &consensus.TokenName{ChainTreeDID: i.inkDID, LocalName: "ink"}

	// TODO: Move all the ink transfer code to its own func
	tokenLedger := consensus.NewTreeLedger(newInviteTree, inkTokenName)

	inkBalance, err := tokenLedger.Balance()
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debugf("moving %d ink to player chaintree", inkBalance)

	tokenPayload, err := i.net.SendInk(newInviteChainTree, inkTokenName, inkBalance, playerChainTree.MustId())
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debug("sent ink to player")

	err = i.net.ReceiveInk(playerChainTree, tokenPayload)
	if err != nil {
		actorCtx.Respond(&inkfaucet.InviteSubmissionResponse{
			Error: err.Error(),
		})
		return
	}

	log.Debug("player chaintree has received ink")

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

