package game

import (
	"github.com/quorumcontrol/jasons-game/game/trees"
)

type ListInteractionsRequest struct{}

type ListInteractionsResponse struct {
	Interactions []trees.Interaction
	Error        error
}

type AddInteractionRequest struct {
	Interaction trees.Interaction
}

type AddInteractionResponse struct {
	Error error
}