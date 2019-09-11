package court

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	broadcastHandlers "github.com/quorumcontrol/jasons-game/handlers/broadcast"
	"github.com/quorumcontrol/jasons-game/importer"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

type prizeConfig struct {
	Location string                 `yaml:"location"`
	Spawn    *importer.ImportObject `yaml:"spawn"`
	Prize    *importer.ImportObject `yaml:"prize"`
}

type PrizeHandlerConfig struct {
	Court           *Court
	PrizeConfigPath string
	ValidatorFunc   func(msg *jasonsgame.RequestObjectTransferMessage) (bool, error)
	CleanupFunc     func(msg *jasonsgame.RequestObjectTransferMessage) error
}

type PrizeHandler struct {
	court         *Court
	net           network.Network
	tree          *consensus.SignedChainTree
	validatorFunc func(msg *jasonsgame.RequestObjectTransferMessage) (bool, error)
	cleanupFunc   func(msg *jasonsgame.RequestObjectTransferMessage) error
	prizeCfg      *prizeConfig
	prizeCfgPath  string
}

var PrizeHandlerMessages = handlers.HandlerMessageList{
	proto.MessageName((*jasonsgame.RequestObjectTransferMessage)(nil)),
}

func NewPrizeHandler(config *PrizeHandlerConfig) (*PrizeHandler, error) {
	handler := &PrizeHandler{
		court:         config.Court,
		net:           config.Court.Network(),
		validatorFunc: config.ValidatorFunc,
		cleanupFunc:   config.CleanupFunc,
		prizeCfgPath:  config.PrizeConfigPath,
	}
	err := handler.setup()
	if err != nil {
		return nil, err
	}
	return handler, nil
}

func (h *PrizeHandler) Tree() *consensus.SignedChainTree {
	return h.tree
}

func (h *PrizeHandler) Name() string {
	return h.court.Name() + "-prize-handler"
}

func (h *PrizeHandler) setup() error {
	var err error
	h.prizeCfg, err = h.parseConfig()
	if err != nil {
		return err
	}

	h.tree, err = h.net.FindOrCreatePassphraseTree(h.Name())
	if err != nil {
		return err
	}

	if h.prizeCfg.Location == "" {
		return errors.Wrap(err, "must set Location in "+h.prizeCfgPath)
	}
	locTree, err := h.net.GetTree(h.prizeCfg.Location)
	if err != nil {
		return errors.Wrap(err, "getting loc tree")
	}
	location := game.NewLocationTree(h.net, locTree)
	err = location.SetHandler(h.tree.MustId())
	if err != nil {
		return errors.Wrap(err, "getting loc tree")
	}

	err = h.spawnObject()
	if err != nil {
		return err
	}

	return nil
}

func (h *PrizeHandler) parseConfig(additionalArgs ...map[string]interface{}) (*prizeConfig, error) {
	vars, err := h.court.Ids()
	if err != nil {
		return nil, errors.Wrap(err, "error fetching ids for court")
	}

	cfg := &prizeConfig{}
	err = config.ReadYaml(h.prizeCfgPath, cfg, append(additionalArgs, vars)...)
	if err != nil {
		return nil, errors.Wrap(err, "error processing "+h.prizeCfgPath)
	}

	return cfg, nil
}

func (h *PrizeHandler) currentObjectDid() (string, error) {
	locTree, err := h.net.GetTree(h.prizeCfg.Location)
	if err != nil {
		return "", errors.Wrap(err, "getting loc tree")
	}

	locInventory := trees.NewInventoryTree(h.net, locTree)

	spawnName := h.prizeCfg.Spawn.Data["name"].(string)

	return locInventory.DidForName(spawnName)
}

func (h *PrizeHandler) currentObjectExists() (bool, error) {
	did, err := h.currentObjectDid()
	return len(did) > 0, err
}

func (h *PrizeHandler) spawnObject() error {
	exists, err := h.currentObjectExists()
	if err != nil {
		return err
	}

	// object still exists, skip
	if exists {
		log.Debugf("prizehandler: skipping spawning new object, already exists at %s", h.prizeCfg.Location)
		return nil
	}

	locTree, err := h.net.GetTree(h.prizeCfg.Location)
	if err != nil {
		return errors.Wrap(err, "getting loc tree")
	}

	// use location tip for deterministically generating the next object so that
	// this can run distributed and stateless
	objectChainTree, err := h.net.FindOrCreatePassphraseTree(locTree.Tip().String())
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

	log.Debugf("prizehandler: new object %s spawned at %s", objectChainTree.MustId(), cfg.Location)

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

func (h *PrizeHandler) handleTransfer(msg *jasonsgame.RequestObjectTransferMessage) error {
	sender := newResponseSender(h.net, msg)

	objectDid := msg.Object
	currentObjectDid, err := h.currentObjectDid()
	if err != nil {
		return sender.Errorf("could not fetch prize did")
	}
	if objectDid != currentObjectDid {
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
	existingPlayerDid, err := playerInventory.DidForName(h.prizeCfg.Prize.Data["name"].(string))
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

	if h.validatorFunc != nil {
		valid, err := h.validatorFunc(msg)
		if err != nil || !valid {
			return sender.Errorf("could not validate: %v", err)
		}
	}

	// Delete object from location inventory
	locTree, err := h.net.GetTree(h.prizeCfg.Location)
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
		log.Error(errors.Wrap(err, "error spawning new object"))
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

	if h.cleanupFunc != nil {
		err = h.cleanupFunc(msg)
		if err != nil {
			log.Error(errors.Wrap(err, "error on cleanup"))
		}
	}

	return sender.Send()
}

func (h *PrizeHandler) Handle(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.RequestObjectTransferMessage:
		err := h.handleTransfer(msg)
		if err != nil {
			log.Errorf("PrizeHandler: %v", err)
		}
		return nil
	default:
		return handlers.ErrUnsupportedMessageType
	}
}

func (h *PrizeHandler) Supports(msg proto.Message) bool {
	return PrizeHandlerMessages.Contains(msg)
}

func (h *PrizeHandler) SupportedMessages() []string {
	return PrizeHandlerMessages
}