package autumn

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	broadcastHandlers "github.com/quorumcontrol/jasons-game/handlers/broadcast"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

const combinationObjectName = "bowl"
const combinationSuccessMsg = "your elements have been combined"
const combinationFailureMsg = "error combining: %s - try again"

const failElementID = 15

const comboObjectPlayerPath = "player"
const comboObjectElementsPath = "elements"

type ElementCombinerHandler struct {
	net             network.Network
	did             string
	location        string
	elements        map[int]*element
	combinationsMap elementCombinationMap
}

type ElementCombinerHandlerConfig struct {
	Did          string
	Network      network.Network
	Location     string
	Elements     []*element
	Combinations []*elementCombination
}

var ElementCombinerHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName((*jasonsgame.TransferredObjectMessage)(nil)),
	proto.MessageName((*jasonsgame.RequestObjectTransferMessage)(nil)),
}

func NewElementCombinerHandler(cfg *ElementCombinerHandlerConfig) (*ElementCombinerHandler, error) {
	h := &ElementCombinerHandler{
		net:             cfg.Network,
		did:             cfg.Did,
		location:        cfg.Location,
		elements:        make(map[int]*element),
		combinationsMap: make(elementCombinationMap),
	}

	h.combinationsMap.Fill(cfg.Combinations)

	for _, element := range cfg.Elements {
		h.elements[element.ID] = element
	}

	err := h.setup()
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *ElementCombinerHandler) setup() error {
	locTree, err := h.net.GetTree(h.location)
	if err != nil {
		return errors.Wrap(err, "getting loc tree")
	}

	// Use a unique inventory per player
	locTree, err = h.net.UpdateChainTree(locTree, "jasons-game/use-per-player-inventory", true)
	if err != nil {
		return errors.Wrap(err, "updating loc tree")
	}

	location := game.NewLocationTree(h.net, locTree)
	err = location.SetHandler(h.did)
	if err != nil {
		return errors.Wrap(err, "getting loc tree")
	}

	return nil
}

func (h *ElementCombinerHandler) handlerAuthentications() ([]string, error) {
	handlerTree, err := h.net.GetTree(h.did)
	if err != nil {
		return nil, err
	}
	return handlerTree.Authentications()
}

type responseSender struct {
	source  *jasonsgame.RequestObjectTransferMessage
	handler handlers.Handler
}

func newResponseSender(net network.Network, source *jasonsgame.RequestObjectTransferMessage) *responseSender {
	remoteTargetHandler, err := handlers.FindHandlerForTree(net, source.To)

	var targetHandler handlers.Handler
	if err != nil || remoteTargetHandler == nil {
		targetHandler = broadcastHandlers.NewTopicBroadcastHandler(net, trees.InventoryBroadcastTopicFor(net, source.To))
	} else {
		targetHandler = remoteTargetHandler
	}

	return &responseSender{
		source:  source,
		handler: targetHandler,
	}
}

func (s *responseSender) Send() error {
	err := s.handler.Handle(&jasonsgame.TransferredObjectMessage{
		From:    s.source.From,
		To:      s.source.To,
		Object:  s.source.Object,
		Message: combinationSuccessMsg,
	})
	if err != nil {
		return errors.Wrap(err, "error transferring object")
	}
	return nil
}

func (s *responseSender) Errorf(str string, args ...interface{}) error {
	err := fmt.Errorf(str, args...)
	handlerErr := s.handler.Handle(&jasonsgame.TransferredObjectMessage{
		From:   s.source.From,
		To:     s.source.To,
		Object: s.source.Object,
		Error:  fmt.Sprintf(combinationFailureMsg, err.Error()),
	})
	if handlerErr != nil {
		err = errors.Wrap(err, handlerErr.Error())
	}
	return err
}

func (h *ElementCombinerHandler) findCombinedElement(ids []int) *element {
	newElementID, ok := h.combinationsMap.Find(ids)
	if !ok {
		newElementID = failElementID
	}
	newElement, ok := h.elements[newElementID]
	if !ok {
		newElement = h.elements[failElementID]
	}
	return newElement
}

func (h *ElementCombinerHandler) handlePickupElement(msg *jasonsgame.RequestObjectTransferMessage) error {
	sender := newResponseSender(h.net, msg)

	inventory, err := trees.FindInventoryTree(h.net, msg.From)
	if err != nil {
		return sender.Errorf("could not fetch current inventory")
	}

	comboDid, err := inventory.DidForName(combinationObjectName)
	if err != nil {
		return sender.Errorf("could not fetch current inventory")
	}

	if comboDid == "" {
		return sender.Errorf("no elements to combine")
	}

	comboObject, err := game.FindObjectTree(h.net, comboDid)
	if err != nil {
		return sender.Errorf("could not fetch object")
	}

	playerDidUncast, err := comboObject.GetPath([]string{comboObjectPlayerPath})
	if err != nil {
		return sender.Errorf("could not fetch object")
	}
	if playerDidUncast == nil || playerDidUncast.(string) != msg.To {
		return sender.Errorf("incorrect player did for object")
	}
	playerTree, err := h.net.GetTree(msg.To)
	if err != nil {
		return sender.Errorf("could not fetch player tree")
	}
	playerAuths, err := playerTree.Authentications()
	if err != nil {
		return sender.Errorf("could not fetch player tree")
	}

	elementsUncast, err := comboObject.GetPath([]string{comboObjectElementsPath})
	if err != nil {
		return sender.Errorf("could not fetch object")
	}
	if elementsUncast == nil {
		return sender.Errorf("no elements to combine")
	}

	elementIds := make([]int, len(elementsUncast.(map[string]interface{})))
	i := 0
	for name := range elementsUncast.(map[string]interface{}) {
		elementIds[i] = elementNameToId(name)
		i++
	}

	if len(elementIds) < 2 {
		return sender.Errorf("you must drop at least 2 objects to be combined")
	}

	newElement := h.findCombinedElement(elementIds)
	log.Debugf("combining %v, result was %v, new name %s", elementIds, newElement.ID, newElement.Name())

	// TODO: check player can't pickup new element becausee one already exists

	err = inventory.Remove(comboObject.MustId())
	if err != nil {
		return sender.Errorf("could not remove object from location")
	}
	// This resets the object back to a vanilla state
	err = comboObject.UpdatePath([]string{}, make(map[string]interface{}))
	if err != nil {
		return sender.Errorf("could not update new element tree")
	}
	comboObject, err = game.CreateObjectOnTree(h.net, newElement.Name(), comboObject.ChainTree())
	if err != nil {
		return sender.Errorf("could not update new element tree")
	}
	err = comboObject.SetDescription(newElement.Description)
	if err != nil {
		return sender.Errorf("could not update new element tree")
	}
	// TODO inscribe origin elements descriptions ordered by id
	err = comboObject.ChangeChainTreeOwner(playerAuths)
	if err != nil {
		return sender.Errorf("could not update new element tree")
	}

	return sender.Send()
}

func (h *ElementCombinerHandler) handleReceiveElement(msg *jasonsgame.TransferredObjectMessage) error {
	// This is a player specific inventory for this location
	targetInventory, err := trees.FindInventoryTree(h.net, msg.To)
	if err != nil {
		return fmt.Errorf("error fetching inventory chaintree: %v", err)
	}

	incomingObject, err := game.FindObjectTree(h.net, msg.Object)
	if err != nil {
		return fmt.Errorf("error fetching object chaintree %s: %v", msg.Object, err)
	}

	// TODO: check player matches
	// TODO: check validity of incoming object
	// TODO: verify this service created all elements, except those spawned in mines

	handlerAuths, err := h.handlerAuthentications()
	if err != nil {
		return err
	}

	existingComboDid, err := targetInventory.DidForName(combinationObjectName)
	if err != nil {
		return err
	}

	var comboObject *game.ObjectTree

	if len(existingComboDid) > 0 {
		comboObject, err = game.FindObjectTree(h.net, existingComboDid)
		if err != nil {
			return err
		}
	} else {
		// use location tip for deterministically generating the next object so that
		// this can run distributed and stateless
		newTree, err := findOrCreateNamedTree(h.net, targetInventory.Tree().Tip().String())
		if err != nil {
			return errors.Wrap(err, "error creating new object key")
		}

		// TODO: customize object interactions
		comboObject, err = game.CreateObjectOnTree(h.net, combinationObjectName, newTree)
		if err != nil {
			return err
		}

		err = comboObject.ChangeChainTreeOwner(handlerAuths)
		if err != nil {
			return err
		}

		err = comboObject.UpdatePath([]string{comboObjectPlayerPath}, msg.From)
		if err != nil {
			return err
		}
	}

	incomingObjectName, err := incomingObject.GetName()
	if err != nil {
		return err
	}

	err = comboObject.UpdatePath([]string{comboObjectElementsPath, incomingObjectName}, incomingObject.MustId())
	if err != nil {
		return err
	}

	err = incomingObject.ChangeChainTreeOwner(handlerAuths)
	if err != nil {
		return fmt.Errorf("error changing object owner: %v", err)
	}

	err = targetInventory.ForceAdd(comboObject.MustId())
	if err != nil {
		return err
	}
	return nil
}

func (h *ElementCombinerHandler) Handle(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.TransferredObjectMessage:
		return h.handleReceiveElement(msg)
	case *jasonsgame.RequestObjectTransferMessage:
		return h.handlePickupElement(msg)
	default:
		return handlers.ErrUnsupportedMessageType
	}
}

func (h *ElementCombinerHandler) Supports(msg proto.Message) bool {
	return ElementCombinerHandlerMessages.Contains(msg)
}

func (h *ElementCombinerHandler) SupportedMessages() []string {
	return ElementCombinerHandlerMessages
}
