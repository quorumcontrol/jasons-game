package game

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

type BarrenWastelandLocationActor struct {
	middleware.LogAwareHolder
	cfg *BarrenWastelandLocationActorConfig
}

type BarrenWastelandLocationActorConfig struct {
	Direction         string
	Network           network.Network
	ReturnInteraction *Interaction
	OnExcavate        func(newDid string)
}

func NewBarrenWastelandLocationActorProps(cfg *BarrenWastelandLocationActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &BarrenWastelandLocationActor{cfg: cfg}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (l *BarrenWastelandLocationActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *GetLocation:
		actorCtx.Respond(&jasonsgame.Location{
			Description: "You are in a barren wasteland type set description <desc> to call this land something",
		})
	case *GetInteraction:
		if msg.Command == l.cfg.ReturnInteraction.Command {
			actorCtx.Respond(l.cfg.ReturnInteraction)
		} else {
			actorCtx.Respond(&Interaction{
				Command: msg.Command,
				Action:  "respond",
				Args: map[string]string{
					"response": "You are in a barren wasteland, you can't go " + msg.Command,
				},
			})
		}
	case *SetLocationDescriptionRequest:
		newTree, err := l.cfg.Network.CreateChainTree()
		if err != nil {
			actorCtx.Respond(&SetLocationDescriptionResponse{Error: errors.Wrap(err, "error creating tree")})
			return
		}

		newLocation := NewLocationTree(l.cfg.Network, newTree)
		err = newLocation.SetDescription(msg.Description)
		if err != nil {
			actorCtx.Respond(&SetLocationDescriptionResponse{Error: errors.Wrap(err, "error updating description on tree")})
		}

		err = newLocation.AddInteraction(l.cfg.ReturnInteraction)
		if err != nil {
			actorCtx.Respond(&SetLocationDescriptionResponse{Error: errors.Wrap(err, "error updating interactions on tree")})
		}

		originalTree, err := l.cfg.Network.GetTree(l.cfg.ReturnInteraction.Args["did"])
		if err != nil {
			actorCtx.Respond(&SetLocationDescriptionResponse{Error: errors.Wrap(err, "error fetching original location")})
		}
		originalLocation := NewLocationTree(l.cfg.Network, originalTree)
		if err != nil {
			actorCtx.Respond(&SetLocationDescriptionResponse{Error: errors.Wrap(err, "error updating description on tree")})
		}

		err = originalLocation.AddInteraction(&Interaction{
			Command: l.cfg.Direction,
			Action:  "changeLocation",
			Args: map[string]string{
				"did": newLocation.MustId(),
			},
		})
		if err != nil {
			actorCtx.Respond(&SetLocationDescriptionResponse{Error: errors.Wrap(err, "error updating interactions on original location")})
		}

		l.cfg.OnExcavate(newLocation.MustId())
	}
}
