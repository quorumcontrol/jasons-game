package summer

import (
	"fmt"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/courts/court"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	broadcastHandlers "github.com/quorumcontrol/jasons-game/handlers/broadcast"
	"github.com/quorumcontrol/jasons-game/importer"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

type prizeConfig struct {
	Location       string                   `yaml:"location"`
	Spawn          *importer.ImportObject   `yaml:"spawn"`
	Prize          *importer.ImportObject   `yaml:"prize"`
	LocationUpdate *importer.ImportLocation `yaml:"location_update"`
}

type SummerPrizeHandler struct {
	net           network.Network
	court         *SummerCourt
	currentObject string
	cfg           *prizeConfig
}

var SummerPrizeHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName((*jasonsgame.RequestObjectTransferMessage)(nil)),
}

func NewSummerPrizeHandler(court *SummerCourt) (*SummerPrizeHandler, error) {
	handler := &SummerPrizeHandler{
		net:   court.net,
		court: court,
	}
	err := handler.setup()
	if err != nil {
		return nil, err
	}
	return handler, nil
}

func (h *SummerPrizeHandler) setup() error {
	var err error
	h.cfg, err = h.parseConfig()
	if err != nil {
		return err
	}

	handlerTree, err := court.FindOrCreateNamedTree(h.net, prizeHandlerTreeName)
	if err != nil {
		return err
	}

	if h.cfg.Location == "" {
		return errors.Wrap(err, "must set Location in prize_config.yml")
	}
	locTree, err := h.net.GetTree(h.cfg.Location)
	if err != nil {
		return errors.Wrap(err, "getting loc tree")
	}
	location := game.NewLocationTree(h.net, locTree)
	err = location.SetHandler(handlerTree.MustId())
	if err != nil {
		return errors.Wrap(err, "getting loc tree")
	}

	err = h.spawnObject()
	if err != nil {
		return err
	}

	return nil
}

func (h *SummerPrizeHandler) parseConfig(additionalArgs ...map[string]interface{}) (*prizeConfig, error) {
	vars, err := h.court.ids()
	if err != nil {
		return nil, errors.Wrap(err, "error fetching ids for court")
	}

	cfg := &prizeConfig{}
	err = config.ReadYaml(filepath.Join(h.court.configPath, "summer/prize_config.yml"), cfg, append(additionalArgs, vars)...)
	if err != nil {
		return nil, errors.Wrap(err, "error processing prize_config.yml")
	}

	return cfg, nil
}

func (h *SummerPrizeHandler) objectExists(name string) (bool, error) {
	var err error

	locTree, err := h.net.GetTree(h.cfg.Location)
	if err != nil {
		return false, errors.Wrap(err, "getting loc tree")
	}

	locInventory := trees.NewInventoryTree(h.net, locTree)

	if h.currentObject == "" {
		h.currentObject, err = locInventory.DidForName(name)

		if err != nil {
			return false, errors.Wrap(err, "error fetching location inventory")
		}
	}

	exists, err := locInventory.Exists(h.currentObject)

	if err != nil {
		return false, errors.Wrap(err, "error checking location inventory")
	}

	return exists, nil
}

func (h *SummerPrizeHandler) spawnObject() error {
	spawnName := h.cfg.Spawn.Data["name"].(string)

	exists, err := h.objectExists(spawnName)
	if err != nil {
		return err
	}

	// object still exists, skip
	if exists {
		log.Debugf("prizehandler: skipping spawning new object, already exists at %s", h.cfg.Location)
		return nil
	}

	locTree, err := h.net.GetTree(h.cfg.Location)
	if err != nil {
		return errors.Wrap(err, "getting loc tree")
	}

	// use location tip for deterministically generating the next object so that
	// this can run distributed and stateless
	objectChainTree, err := court.FindOrCreateNamedTree(h.net, locTree.Tip().String())
	if err != nil {
		return errors.Wrap(err, "error creating new object key")
	}

	cfg, err := h.parseConfig(map[string]interface{}{"PrizeDid": objectChainTree.MustId()})
	if err != nil {
		return err
	}

	err = importer.New(h.net).UpdateObject(objectChainTree.MustId(), cfg.Spawn)
	if err != nil {
		return err
	}

	locInventory := trees.NewInventoryTree(h.net, locTree)
	err = locInventory.Add(objectChainTree.MustId())
	if err != nil {
		return err
	}

	h.currentObject = objectChainTree.MustId()
	log.Debugf("prizehandler: new object %s spawned at %s", h.currentObject, cfg.Location)

	return nil
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
	log.Debugf("prizehandler: sending prize %s to %s", s.source.To, s.source.Object)
	err := s.handler.Handle(&jasonsgame.TransferredObjectMessage{
		From:    s.source.From,
		To:      s.source.To,
		Object:  s.source.Object,
		Message: " ",
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
		Error:  fmt.Sprintf("error on pickup: %s - try again", err.Error()),
	})
	if handlerErr != nil {
		err = errors.Wrap(err, handlerErr.Error())
	}
	return err
}

func (h *SummerPrizeHandler) handleTransfer(msg *jasonsgame.RequestObjectTransferMessage) error {
	sender := newResponseSender(h.net, msg)

	objectDid := msg.Object
	if objectDid != h.currentObject {
		return sender.Errorf("current object has changed")
	}

	// check player inventory doesn't already have prize
	playerTree, err := h.net.GetTree(msg.To)
	if err != nil {
		return sender.Errorf("could not fetch player chaintree")
	}
	playerInventory, err := trees.FindInventoryTree(h.net, playerTree.MustId())
	if err != nil {
		return sender.Errorf("could not fetch player inventory")
	}
	existingPlayerDid, err := playerInventory.DidForName(h.cfg.Prize.Data["name"].(string))
	if err != nil {
		return sender.Errorf("fetching object in player inventory")
	}
	exists, err := playerInventory.Exists(existingPlayerDid)
	if err != nil {
		return sender.Errorf("checking existance in player inventory")
	}
	if exists {
		return sender.Errorf("prize already exists in player inventory")
	}

	// Delete object from location inventory
	locTree, err := h.net.GetTree(h.cfg.Location)
	if err != nil {
		return sender.Errorf("could not fetch location tree")
	}
	locInventory := trees.NewInventoryTree(h.net, locTree)
	err = locInventory.Remove(objectDid)
	if err != nil {
		return sender.Errorf("could not remove object from location")
	}

	// Spawn new object
	err = h.spawnObject()
	if err != nil {
		log.Errorf(errors.Wrap(err, "error spawning new object").Error())
		defer func() {
			panic("panic on respawn")
		}()
	}

	// Update object to send from wire => prize
	cfg, err := h.parseConfig(map[string]interface{}{"PrizeDid": objectDid})
	if err != nil {
		return sender.Errorf("generating prize chaintree failed")
	}
	objectTree, err := h.net.GetTree(objectDid)
	if err != nil {
		return sender.Errorf("generating prize chaintree failed")
	}
	objectTree, err = h.net.UpdateChainTree(objectTree, "jasons-game", make(map[string]interface{}))
	if err != nil {
		return sender.Errorf("generating prize chaintree failed")
	}
	err = importer.New(h.net).UpdateObject(objectTree.MustId(), cfg.Prize)
	if err != nil {
		return sender.Errorf("generating prize chaintree failed")
	}

	// Change owner to player, GetTree required for refresh
	objectTree, err = h.net.GetTree(objectDid)
	if err != nil {
		return sender.Errorf("generating prize chaintree failed")
	}
	playerAuths, err := playerTree.Authentications()
	if err != nil {
		return sender.Errorf("could not fetch player authentications")
	}
	_, err = h.net.ChangeChainTreeOwner(objectTree, playerAuths)
	if err != nil {
		return sender.Errorf("could not update object ownership")
	}

	return sender.Send()
}

func (h *SummerPrizeHandler) Handle(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.RequestObjectTransferMessage:
		err := h.handleTransfer(msg)
		if err != nil {
			log.Errorf("SummerPrizeHandler: %v", err)
		}
		return nil
	default:
		return handlers.ErrUnsupportedMessageType
	}
}

func (h *SummerPrizeHandler) Supports(msg proto.Message) bool {
	return SummerPrizeHandlerMessages.Contains(msg)
}

func (h *SummerPrizeHandler) SupportedMessages() []string {
	return SummerPrizeHandlerMessages
}
