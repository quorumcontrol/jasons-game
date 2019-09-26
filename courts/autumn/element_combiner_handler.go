package autumn

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	broadcastHandlers "github.com/quorumcontrol/jasons-game/handlers/broadcast"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

const combinationObjectName = "bowl"
const combinationSuccessMsg = "Your offering has been accepted, a new element is now yours."
const combinationBaseFailureMsg = "error combining: %s - try again"
const combinationFailureMsg = "The %s gave you nothing for your offering. It must not have been deemed acceptable."
const combinationBlockedFailureMsg = "The Fae are especially susceptible to silver therefore transmuting elements into silver can not be allowed. Your offering has not been deemed worthy."
const combinationNumFailureMsg = "A proper offering must include %d elements."

const comboObjectPlayerPath = "player"
const comboObjectElementsPath = "elements"

type ElementCombinerHandler struct {
	net             network.Network
	tree            *consensus.SignedChainTree
	name            string
	location        string
	elements        map[int]*element
	combinationsMap elementCombinationMap
	minRequired     int
}

type ElementCombinerHandlerConfig struct {
	Name         string
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
		name:            cfg.Name,
		location:        cfg.Location,
		elements:        make(map[int]*element),
		combinationsMap: make(elementCombinationMap),
	}

	h.combinationsMap.Fill(cfg.Combinations)

	h.minRequired = len(cfg.Combinations[0].From)
	for _, combo := range cfg.Combinations {
		if len(combo.From) < h.minRequired {
			h.minRequired = len(combo.From)
		}
	}

	for _, element := range cfg.Elements {
		h.elements[element.ID] = element
	}

	err := h.setup()
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *ElementCombinerHandler) Name() string {
	return h.name
}

func (h *ElementCombinerHandler) Tree() *consensus.SignedChainTree {
	return h.tree
}

func (h *ElementCombinerHandler) setup() error {
	var err error
	h.tree, err = h.net.FindOrCreatePassphraseTree("element-combiner-" + h.Name())
	if err != nil {
		return err
	}

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
	err = location.SetHandler(h.Tree().MustId())
	if err != nil {
		return errors.Wrap(err, "getting loc tree")
	}

	return nil
}

func (h *ElementCombinerHandler) handlerAuthentications() ([]string, error) {
	return h.tree.Authentications()
}

func (h *ElementCombinerHandler) originAuthentications() []string {
	return []string{crypto.PubkeyToAddress(*h.net.PublicKey()).String()}
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
		Error:  err.Error(),
	})
	if handlerErr != nil {
		err = errors.Wrap(err, handlerErr.Error())
	}
	return err
}

func baseErr(str string, args ...interface{}) string {
	return fmt.Sprintf(combinationBaseFailureMsg, fmt.Sprintf(str, args...))
}

func (h *ElementCombinerHandler) findCombinedElement(ids []int) *element {
	newElementID, ok := h.combinationsMap.Find(ids)
	if !ok {
		return nil
	}
	newElement, ok := h.elements[newElementID]
	if !ok {
		return nil
	}
	return newElement
}

func (h *ElementCombinerHandler) handlePickupElement(msg *jasonsgame.RequestObjectTransferMessage) error {
	log.Debugf("handlePickupElement: received RequestObjectTransferMessage: from=%s to=%s obj=%s", msg.From, msg.To, msg.Object)

	sender := newResponseSender(h.net, msg)

	inventory, err := trees.FindInventoryTree(h.net, msg.From)
	if err != nil {
		return sender.Errorf(baseErr("could not fetch current inventory"))
	}

	comboDid, err := inventory.DidForName(combinationObjectName)
	if err != nil {
		return sender.Errorf(baseErr("could not fetch current inventory"))
	}

	if comboDid == "" {
		log.Debugf("handlePickupElement: no combination object not in inventory: from=%s obj=%s", msg.From, msg.Object)
		return sender.Errorf(baseErr("no elements to combine"))
	}

	if comboDid != msg.Object {
		log.Debugf("handlePickupElement: wrong combination object in inventory: from=%s obj=%s combo=%s", msg.From, msg.Object, comboDid)
		return sender.Errorf(baseErr("wrong object did for %s", combinationObjectName))
	}

	comboObject, err := game.FindObjectTree(h.net, comboDid)
	if err != nil {
		return sender.Errorf(baseErr("could not fetch object"))
	}

	playerDidUncast, err := comboObject.GetPath([]string{comboObjectPlayerPath})
	if err != nil {
		return sender.Errorf(baseErr("could not fetch object"))
	}
	if playerDidUncast == nil || playerDidUncast.(string) != msg.To {
		return sender.Errorf(baseErr("incorrect player did for object"))
	}
	playerTree, err := h.net.GetTree(msg.To)
	if err != nil {
		return sender.Errorf(baseErr("could not fetch player tree"))
	}
	playerAuths, err := playerTree.Authentications()
	if err != nil {
		return sender.Errorf(baseErr("could not fetch player tree"))
	}
	playerInventory := trees.NewInventoryTree(h.net, playerTree)

	elementsUncast, err := comboObject.GetPath([]string{comboObjectElementsPath})
	if err != nil {
		return sender.Errorf(baseErr("could not fetch object"))
	}
	if elementsUncast == nil {
		return sender.Errorf(baseErr("no elements to combine"))
	}

	elementIds := make([]int, len(elementsUncast.(map[string]interface{})))
	i := 0
	for name := range elementsUncast.(map[string]interface{}) {
		elementIds[i] = elementNameToId(name)
		i++
	}

	if len(elementIds) < h.minRequired {
		log.Debugf("handlePickupElement: not enough element: obj=%s", msg.Object)
		return sender.Errorf(combinationNumFailureMsg, h.minRequired)
	}

	newElement := h.findCombinedElement(elementIds)
	// if nil, its a failed combo
	if newElement == nil {
		log.Debugf("handlePickupElement: combining failed: obj=%s elementIds=%v", msg.Object, elementIds)
		err = inventory.Remove(comboObject.MustId())
		if err != nil {
			log.Errorf("could not remove object from location")
		}
		return sender.Errorf(combinationFailureMsg, strings.Title(h.Name()))
	}

	log.Debugf("handlePickupElement: combining: obj=%s elementIds=%v newElement=%s", msg.Object, elementIds, newElement.Name())

	if newElement.ID == -1 {
		err = inventory.Remove(comboObject.MustId())
		if err != nil {
			log.Errorf("could not remove object from location")
		}
		return sender.Errorf(combinationBlockedFailureMsg)
	}

	existing, err := playerInventory.DidForName(newElement.Name())
	if err != nil {
		return sender.Errorf(baseErr("could not fetch player inventory"))
	}
	// if player already has object, return error
	if len(existing) > 0 {
		return sender.Errorf(baseErr("can not pick up %s, one already exists in your inventory", newElement.Name()))
	}

	err = inventory.Remove(comboObject.MustId())
	if err != nil {
		return sender.Errorf(baseErr("could not remove object from location"))
	}
	// This resets the object back to a vanilla state
	err = comboObject.UpdatePath([]string{}, make(map[string]interface{}))
	if err != nil {
		return sender.Errorf(baseErr("could not update new element tree"))
	}
	comboObject, err = game.CreateObjectOnTree(h.net, newElement.Name(), comboObject.ChainTree())
	if err != nil {
		return sender.Errorf(baseErr("could not update new element tree"))
	}
	err = comboObject.SetDescription(newElement.Description)
	if err != nil {
		return sender.Errorf(baseErr("could not update new element tree"))
	}
	// TODO inscribe origin elements descriptions ordered by id

	err = comboObject.ChangeChainTreeOwner(playerAuths)
	if err != nil {
		return sender.Errorf(baseErr("could not update new element tree"))
	}

	return sender.Send()
}

func (h *ElementCombinerHandler) isValidElement(object *game.ObjectTree) (bool, error) {
	elementName, err := object.GetName()
	if err != nil {
		return false, err
	}
	elementID := elementNameToId(elementName)
	log.Debugf("isValidElement: starting validation: obj=%s name=%s id=%d", object.MustId(), elementName, elementID)

	element, ok := h.elements[elementID]
	if !ok {
		log.Debugf("isValidElement: element not found: obj=%s", object.MustId())
		return false, fmt.Errorf("element %d not found", elementID)
	}
	if element.SkipOriginValidation {
		log.Debugf("isValidElement: skpping origin validation: obj=%s", object.MustId())
		return true, nil
	}

	return validateElementOrigin(object, h.originAuthentications())
}

func (h *ElementCombinerHandler) handleReceiveElement(msg *jasonsgame.TransferredObjectMessage) error {
	ctx := context.Background()
	log.Debugf("handleReceiveElement: received TransferredObjectMessage: from=%s to=%s obj=%s", msg.From, msg.To, msg.Object)

	// This is a player specific inventory for this location
	targetInventory, err := trees.FindInventoryTree(h.net, msg.To)
	if err != nil {
		return fmt.Errorf("error fetching inventory chaintree: %v", err)
	}

	incomingObject, err := game.FindObjectTree(h.net, msg.Object)
	if err != nil {
		return fmt.Errorf("error fetching object chaintree %s: %v", msg.Object, err)
	}

	isValid, err := h.isValidElement(incomingObject)
	if err != nil {
		log.Error(err)
		return err
	}
	// Player tried to hack the object, lets destroy it
	if !isValid {
		log.Debugf("handleReceiveElement: object is NOT valid, destroying it: obj=%s", msg.Object)

		err = incomingObject.ChangeChainTreeOwner([]string{})
		if err != nil {
			return fmt.Errorf("error destroying object: %v", err)
		}

		return nil
	}
	log.Debugf("handleReceiveElement: object is valid, continuing: obj=%s", msg.Object)

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
		log.Debugf("handleReceiveElement: found existing combo object: obj=%s comboObj=%s", msg.Object, existingComboDid)
		comboObject, err = game.FindObjectTree(h.net, existingComboDid)
		if err != nil {
			return err
		}

		validOrigin, err := trees.VerifyOwnershipAt(ctx, comboObject.ChainTree().ChainTree, 0, h.originAuthentications())
		if err != nil {
			return err
		}
		if !validOrigin {
			log.Debugf("handleReceiveElement: existing combo object is invalid, will create new one")
			comboObject = nil
		}
	}

	if comboObject == nil {
		// use location tip for deterministically generating the next object so that
		// this can run distributed and stateless
		newTree, err := court.FindOrCreateNamedTree(h.net, targetInventory.Tree().Tip().String())
		if err != nil {
			return errors.Wrap(err, "error creating new object key")
		}
		log.Debugf("handleReceiveElement: created new combo object: obj=%s comboObj=%s", msg.Object, newTree.MustId())

		// Sometimes a tree can be created, but not fully make it to adding to inventory,
		// which gives it some corrupt / inflight state, if thats the case, reset it
		if trees.MustHeight(ctx, newTree.ChainTree) > 1 {
			newTree, err = h.net.UpdateChainTree(newTree, "", make(map[string]interface{}))
			if err != nil {
				return errors.Wrap(err, "error reset new object key")
			}
		}

		comboObject = game.NewObjectTree(h.net, newTree)

		err = comboObject.SetName(combinationObjectName)
		if err != nil {
			return err
		}

		err = comboObject.AddInteraction(&game.PickUpObjectInteraction{
			Command: "submit offering",
			Did:     comboObject.MustId(),
		})
		if err != nil {
			return err
		}

		err = comboObject.AddInteraction(&game.GetTreeValueInteraction{
			Command: "look at " + combinationObjectName,
			Did:     comboObject.MustId(),
			Path:    "description",
		})
		if err != nil {
			return err
		}

		err = comboObject.SetDescription("inside the bowl you have prepared:")
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

	log.Debugf("handleReceiveElement: added incoming %s object to combo: obj=%s comboObj=%s", incomingObjectName, msg.Object, comboObject.MustId())

	err = comboObject.UpdatePath([]string{comboObjectElementsPath, incomingObjectName}, incomingObject.MustId())
	if err != nil {
		return err
	}

	description, err := comboObject.GetDescription()
	if err != nil {
		return err
	}
	err = comboObject.SetDescription(description + "\n  > " + incomingObjectName)
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
	log.Debugf("handleReceiveElement: combo object has been updated: obj=%s comboObj=%s", msg.Object, comboObject.MustId())

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
