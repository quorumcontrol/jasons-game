package game

import (
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

type LocationActor struct {
	middleware.LogAwareHolder
	did            string
	location       *LocationTree
	network        network.Network
	inventoryActor *actor.PID
}

type LocationActorConfig struct {
	Network network.Network
	Did     string
}

type GetLocation struct{}

type SetLocationDescriptionRequest struct {
	Description string
}

type SetLocationDescriptionResponse struct {
	Error error
}

type BuildPortalRequest struct {
	To string
}

type BuildPortalResponse struct {
	Error error
}

func NewLocationActorProps(cfg *LocationActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &LocationActor{
			did:     cfg.Did,
			network: cfg.Network,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (l *LocationActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		tree, err := l.network.GetTree(l.did)
		if err != nil {
			panic("could not find location")
		}
		l.location = NewLocationTree(l.network, tree)

		actorCtx.Spawn(l.network.NewCurrentStateSubscriptionProps(l.did))

		_, err = l.network.Community().SubscribeActor(actorCtx.Self(), l.network.Community().TopicFor(l.did))
		if err != nil {
			panic(errors.Wrap(err, "error spawning land actor subscription"))
		}

		l.inventoryActor = actorCtx.Spawn(NewInventoryActorProps(&InventoryActorConfig{
			Did:     l.did,
			Network: l.network,
		}))
	case *GetLocation:
		desc, err := l.location.GetDescription()
		if err != nil {
			panic(errors.Wrap(err, "error getting description"))
		}

		portal, err := l.location.GetPortal()
		if err != nil {
			panic(errors.Wrap(err, "error getting portal"))
		}

		actorCtx.Respond(&jasonsgame.Location{
			Did:         l.location.MustId(),
			Tip:         l.location.Tip().String(),
			Description: desc,
			Portal:      portal,
		})
	case *SetLocationDescriptionRequest:
		err := l.location.SetDescription(msg.Description)
		actorCtx.Respond(&SetLocationDescriptionResponse{Error: err})
	case *InventoryListRequest:
		actorCtx.Forward(l.inventoryActor)
	case *TransferObjectRequest:
		actorCtx.Forward(l.inventoryActor)
	case *BuildPortalRequest:
		l.handleBuildPortal(actorCtx, msg)
	case *AddInteractionRequest:
		actorCtx.Respond(&AddInteractionResponse{
			Error: l.location.AddInteraction(msg.Interaction),
		})
	case *ListInteractionsRequest:
		l.handleListInteractionsRequest(actorCtx, msg)
	case *signatures.CurrentState:
		if parentPID := actorCtx.Parent(); parentPID != nil {
			actorCtx.Send(parentPID, &StateChange{PID: actorCtx.Self()})
		}
	}
}

func (l *LocationActor) handleListInteractionsRequest(actorCtx actor.Context, msg *ListInteractionsRequest) {
	interactions, err := l.location.InteractionsList()

	if err != nil {
		actorCtx.Respond(&ListInteractionsResponse{Error: err})
		return
	}

	portal, err := l.location.GetPortal()
	if err != nil {
		actorCtx.Respond(&ListInteractionsResponse{Error: errors.Wrap(err, "error getting portal")})
		return
	}

	if portal != nil {
		interactions = append(interactions, &ChangeLocationInteraction{
			Command: "go through portal",
			Did:     portal.To,
		})
	}

	inventoryInteractionsResp, err := actorCtx.RequestFuture(l.inventoryActor, &ListInteractionsRequest{}, 30*time.Second).Result()
	if err != nil {
		actorCtx.Respond(&ListInteractionsResponse{Error: err})
		return
	}

	if inventoryInteractionsResp != nil {
		inventoryInteractions, ok := inventoryInteractionsResp.(*ListInteractionsResponse)
		if !ok {
			actorCtx.Respond(&ListInteractionsResponse{Error: err})
			return
		}

		if inventoryInteractions.Error != nil {
			actorCtx.Respond(&ListInteractionsResponse{Error: inventoryInteractions.Error})
			return
		}

		interactions = append(interactions, inventoryInteractions.Interactions...)
	}

	actorCtx.Respond(&ListInteractionsResponse{
		Interactions: interactions,
		Error:        err,
	})
}

func (l *LocationActor) handleBuildPortal(actorCtx actor.Context, msg *BuildPortalRequest) {
	if msg.To == "" {
		actorCtx.Respond(&BuildPortalResponse{Error: fmt.Errorf("must specify a did to build a portal")})
		return
	}

	err := l.location.BuildPortal(msg.To)
	if err != nil {
		actorCtx.Respond(&BuildPortalResponse{Error: err})
		return
	}

	actorCtx.Respond(&BuildPortalResponse{})
}

func (l *LocationActor) SignedTree() *consensus.SignedChainTree {
	tree, err := l.network.GetTree(l.did)
	if err != nil {
		panic(fmt.Errorf("could not find chaintree with did %v", l.did))
	}
	return tree
}
