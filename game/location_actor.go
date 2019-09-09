package game

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
)

type LocationActor struct {
	middleware.LogAwareHolder
	did            string
	playerDid      string
	location       *LocationTree
	network        network.Network
	inventoryActor *actor.PID
	inventoryDid   string
}

type LocationActorConfig struct {
	Network   network.Network
	Did       string
	PlayerDid string
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

type DeletePortalRequest struct {
}

type DeletePortalResponse struct {
	Error error
}

type GetInventoryDid struct{}

func NewLocationActorProps(cfg *LocationActorConfig) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &LocationActor{
			did:       cfg.Did,
			network:   cfg.Network,
			playerDid: cfg.PlayerDid,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (l *LocationActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		l.initialize(actorCtx)
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
	case *DeletePortalRequest:
		actorCtx.Respond(&DeletePortalResponse{
			Error: l.location.DeletePortal(),
		})
	case *AddInteractionRequest:
		actorCtx.Respond(&AddInteractionResponse{
			Error: l.location.AddInteraction(msg.Interaction),
		})
	case *ListInteractionsRequest:
		l.handleListInteractionsRequest(actorCtx, msg)
	case *GetInventoryDid:
		actorCtx.Respond(l.inventoryDid)
	case *StateChange:
		// Forward inventory actor changes when using different tree
		// when using the same tree, CurrentState below will trigger
		parentPID := actorCtx.Parent()
		if parentPID != nil && l.did != l.inventoryDid {
			actorCtx.Send(parentPID, &StateChange{PID: actorCtx.Self()})
		}
	case *signatures.CurrentState:
		if string(msg.Signature.ObjectId) == l.did {
			newTip, err := cid.Cast(msg.Signature.NewTip)
			if err != nil {
				l.Log.Error(errors.Wrap(err, "could not refresh location"))
				return
			}
			refreshedLocation, err := l.network.GetTreeByTip(newTip)
			if err != nil {
				l.Log.Error(errors.Wrap(err, "could not refresh location"))
				return
			}

			err = l.network.TreeStore().UpdateTreeMetadata(refreshedLocation)
			if err != nil {
				l.Log.Error(errors.Wrap(err, "could not refresh location"))
				return
			}

			l.location = NewLocationTree(l.network, refreshedLocation)
		}
		if parentPID := actorCtx.Parent(); parentPID != nil {
			actorCtx.Send(parentPID, &StateChange{PID: actorCtx.Self()})
		}
	}
}

func (l *LocationActor) initialize(actorCtx actor.Context) {
	tree, err := l.network.GetTree(l.did)
	if err != nil {
		panic(errors.Wrap(err, "error fetching location"))
	}
	if tree == nil {
		panic("could not find location " + l.did)
	}

	l.location = NewLocationTree(l.network, tree)

	actorCtx.Spawn(l.network.NewCurrentStateSubscriptionProps(l.did))

	_, err = l.network.Community().SubscribeActor(actorCtx.Self(), l.network.Community().TopicFor(l.did))
	if err != nil {
		panic(errors.Wrap(err, "error spawning land actor subscription"))
	}

	err = l.spawnInventoryActor(actorCtx)
	if err != nil {
		panic(errors.Wrap(err, "error spawning inventory actor"))
	}
}

func (l *LocationActor) spawnInventoryActor(actorCtx actor.Context) error {
	// default inventory embbedded inside location
	l.inventoryDid = l.did

	usePerPlayerInventory, _, err := l.location.tree.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "use-per-player-inventory"})

	if err != nil {
		return errors.Wrap(err, "error fetching location inventory")
	}

	inventoryLookupPath := []string{"tree", "data", "jasons-game", "location-inventories", l.did}

	if usePerPlayerInventory != nil && usePerPlayerInventory.(bool) {
		playerTree, err := l.network.GetTree(l.playerDid)
		if err != nil {
			return errors.Wrap(err, "error fetching player chain")
		}

		inventoryDidUncast, _, err := playerTree.ChainTree.Dag.Resolve(context.Background(), inventoryLookupPath)
		if err != nil {
			return errors.Wrap(err, "error fetching player chain data")
		}

		var inventoryTree *consensus.SignedChainTree

		if inventoryDidUncast != nil && inventoryDidUncast.(string) != "" {
			inventoryTree, err = l.network.GetChainTreeByName(inventoryDidUncast.(string))

			if err != nil {
				return errors.Wrap(err, "error fetching player inventory for location")
			}
		}

		if inventoryTree == nil {
			inventoryTree, err = l.network.CreateChainTree()
			if err != nil {
				return errors.Wrap(err, "error creating inventory tree")
			}

			inventoryTree, err = l.network.UpdateChainTree(inventoryTree, "jasons-game/inventory-for", l.did)
			if err != nil {
				return errors.Wrap(err, "error updating inventory tree")
			}

			inventoryAuths, err := inventoryTree.Authentications()
			if err != nil {
				return errors.Wrap(err, "error fetching inventory auths")
			}

			locationHandler, err := handlers.FindHandlerForTree(l.network, l.did)
			if err != nil {
				return errors.Wrap(err, "error fetching inventory handler")
			}

			var additionalAuths []string

			if locationHandler != nil {
				handlerTree, err := l.network.GetTree(locationHandler.Did())
				if err != nil {
					return errors.Wrap(err, "error fetching handler tree")
				}

				additionalAuths, err = handlerTree.Authentications()
				if err != nil {
					return errors.Wrap(err, "error fetching handler auths")
				}

				inventoryTree, err = l.network.UpdateChainTree(inventoryTree, "jasons-game-handler", handlerTree.MustId())
				if err != nil {
					return errors.Wrap(err, "error setting new handler attr")
				}
			} else {
				additionalAuths, err = l.location.tree.Authentications()
				if err != nil {
					return errors.Wrap(err, "error fetching location auths")
				}
			}

			inventoryTree, err = l.network.ChangeChainTreeOwner(inventoryTree, append(inventoryAuths, additionalAuths...))
			if err != nil {
				return errors.Wrap(err, "error setting new handler auths")
			}

			_, err = l.network.UpdateChainTree(playerTree, strings.Join(inventoryLookupPath[2:], "/"), inventoryTree.MustId())
			if err != nil {
				return errors.Wrap(err, "error updating player tree")
			}
		}

		l.inventoryDid = inventoryTree.MustId()
	}

	l.inventoryActor = actorCtx.Spawn(NewInventoryActorProps(&InventoryActorConfig{
		Did:     l.inventoryDid,
		Network: l.network,
	}))
	return nil
}

func (l *LocationActor) handleListInteractionsRequest(actorCtx actor.Context, msg *ListInteractionsRequest) {
	localKeyAddr := consensus.DidToAddr(consensus.EcdsaPubkeyToDid(*l.network.PublicKey()))
	isLocal, err := l.location.IsOwnedBy([]string{localKeyAddr})
	if err != nil {
		actorCtx.Respond(&ListInteractionsResponse{Error: errors.Wrap(err, "error getting owner auths")})
		return
	}

	interactions := []*InteractionResponse{}

	if isLocal {
		interactions = append(interactions, &InteractionResponse{
			AttachedTo:    "location",
			AttachedToDid: l.did,
			Interaction: &SetTreeValueInteraction{
				Command: "set description",
				Did:     l.location.MustId(),
				Path:    "description",
			},
		})
	}

	portal, err := l.location.GetPortal()
	if err != nil {
		actorCtx.Respond(&ListInteractionsResponse{Error: errors.Wrap(err, "error getting portal")})
		return
	}

	if portal != nil {
		interactions = append(interactions, &InteractionResponse{
			AttachedTo:    "location",
			AttachedToDid: l.did,
			Interaction: &ChangeLocationInteraction{
				Command: "go through portal",
				Did:     portal.To,
			},
		})

		if isLocal {
			interactions = append(interactions, &InteractionResponse{
				AttachedTo:    "location",
				AttachedToDid: l.did,
				Interaction:   &DeletePortalInteraction{},
			})
		}
	} else {
		if isLocal {
			interactions = append(interactions, &InteractionResponse{
				AttachedTo:    "location",
				AttachedToDid: l.did,
				Interaction:   &BuildPortalInteraction{},
			})
		}
	}

	locInteractions, err := l.location.InteractionsList()
	if err != nil {
		actorCtx.Respond(&ListInteractionsResponse{Error: err})
		return
	}

	for _, interaction := range locInteractions {
		interactions = append(interactions, &InteractionResponse{
			AttachedTo:    "location",
			AttachedToDid: l.did,
			Interaction:   interaction,
		})
	}

	// appending this after interactions so location can overwrite
	// the `look around` command
	interactions = append(interactions, &InteractionResponse{
		AttachedTo:    "location",
		AttachedToDid: l.did,
		Interaction:   &LookAroundInteraction{},
	})

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
