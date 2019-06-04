package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

// jasonsgame/interactions/go/north

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

type ListInteractionsRequest struct{}

type ListInteractionsResponse struct {
	Interactions []string
	Error        error
}

type GetInteractionRequest struct {
	Command string
}

type GetInteractionResponse struct {
	Interaction *Interaction
	Error       error
}

type AddInteractionRequest struct {
	Interaction *Interaction
}

type AddInteractionResponse struct {
	Error error
}

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

		_, err = l.network.Community().SubscribeActor(actorCtx.Self(), topicFor(l.did))
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
			panic(fmt.Errorf("error getting description: %v", err))
		}

		portal, err := l.location.GetPortal()
		if err != nil {
			panic(fmt.Errorf("error getting portal: %v", err))
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
	case *GetInteractionRequest:
		l.handleGetInteractionRequest(actorCtx, msg)
	case *ListInteractionsRequest:
		l.handleListInteractionsRequest(actorCtx, msg)
	}
}

func (l *LocationActor) handleGetInteractionRequest(actorCtx actor.Context, msg *GetInteractionRequest) {
	// TODO: allow portal to expose interactions
	if msg.Command == "go through portal" {
		portal, err := l.location.GetPortal()
		if err != nil {
			actorCtx.Respond(&GetInteractionResponse{Error: fmt.Errorf("error getting portal: %v", err)})
			return
		}
		actorCtx.Respond(&Interaction{
			Command: msg.Command,
			Action:  "changeLocation",
			Args: map[string]string{
				"did": portal.To,
			},
		})
		return
	}

	interaction, err := l.location.GetInteraction(msg.Command)
	if err != nil {
		panic(errors.Wrap(err, "error fetching interaction"))
	}
	actorCtx.Respond(interaction)
}

func (l *LocationActor) handleListInteractionsRequest(actorCtx actor.Context, msg *ListInteractionsRequest) {
	interactions, err := l.location.InteractionsList()
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
