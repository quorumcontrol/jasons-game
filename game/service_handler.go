package game

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/protobuf/proto"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

const HandlerPath = "jasons-game-handler"

type Handler interface {
	Send(proto.Message) error
	Supports(proto.Message) bool
	SupportsType(string) bool
}

type LocalHandler struct {
	network network.Network
}

func NewLocalHandler(net network.Network) *LocalHandler {
	return &LocalHandler{network: net}
}

func (h *LocalHandler) Send(msg proto.Message) error {
	switch msg := msg.(type) {
	case *jasonsgame.TransferObjectMessage:
		sourceInventory, err := FindInventoryTree(h.network, msg.From)
		if err != nil {
			return fmt.Errorf("error fetching source chaintree: %v", err)
		}

		targetTree, err := h.network.GetTree(msg.To)
		if err != nil {
			return err
		}

		targetAuths, err := targetTree.Authentications()
		if err != nil {
			return fmt.Errorf("error fetching target chaintree authentications %s; error: %v", msg.To, err)
		}

		objectTree, err := FindObjectTree(h.network, msg.Object)
		if err != nil {
			return fmt.Errorf("error fetching object chaintree %s: %v", msg.Object, err)
		}

		remoteTargetHandler, err := FindHandlerForTree(h.network, msg.To)
		if err != nil {
			return fmt.Errorf("error fetching handler for %v", msg.To)
		}
		var targetHandler Handler
		if remoteTargetHandler != nil {
			targetHandler = remoteTargetHandler
		} else {
			targetHandler = NewLocalHandler(h.network)
		}

		transferredObjectMessage := &jasonsgame.TransferredObjectMessage{
			From:   msg.From,
			To:     msg.To,
			Object: msg.Object,
		}

		if !targetHandler.Supports(transferredObjectMessage) {
			return fmt.Errorf("transfer to inventory %v is not supported", msg.To)
		}

		err = objectTree.ChangeOwner(targetAuths)
		if err != nil {
			return err
		}

		err = sourceInventory.Remove(msg.Object)
		if err != nil {
			return fmt.Errorf("error updating objects in inventory: %v", err)
		}

		if err := targetHandler.Send(transferredObjectMessage); err != nil {
			return err
		}

		return nil
	case *jasonsgame.TransferredObjectMessage:
		actorCtx := actor.EmptyRootContext
		tempInvActor := actorCtx.Spawn(NewInventoryActorProps(&InventoryActorConfig{
			Did:     msg.To,
			Network: h.network,
		}))
		actorCtx.Send(tempInvActor, msg)
		actorCtx.Poison(tempInvActor)
	}
	return nil
}

func (h *LocalHandler) Supports(msg proto.Message) bool {
	return true
}

func (h *LocalHandler) SupportsType(msgType string) bool {
	return true
}

type RemoteHandler struct {
	did               string
	net               network.Network
	supportedMessages map[string]bool
}

func (h *RemoteHandler) Send(msg proto.Message) error {
	if !h.Supports(msg) {
		return fmt.Errorf("handler does not support message %v", msg)
	}
	return h.net.Community().Send(topicFor(h.did), msg)
}

func (h *RemoteHandler) Supports(msg proto.Message) bool {
	msgType := proto.MessageName(msg)
	return h.SupportsType(msgType)
}

func (h *RemoteHandler) SupportsType(msgType string) bool {
	return h.supportedMessages[msgType] || false
}

func FindHandlerForTree(net network.Network, did string) (*RemoteHandler, error) {
	tree, err := net.GetTree(did)
	if err != nil {
		return nil, err
	}

	handlerDid, _, err := tree.ChainTree.Dag.Resolve([]string{"tree", "data", HandlerPath})
	if err != nil {
		return nil, err
	}

	if handlerDid == nil {
		return nil, nil
	}

	handlerTree, err := net.GetTree(handlerDid.(string))
	if err != nil {
		return nil, err
	}

	supports, _, err := handlerTree.ChainTree.Dag.Resolve([]string{"tree", "data", "jasons-game", "supports"})
	if err != nil {
		return nil, err
	}
	if supports == nil {
		supports = []interface{}{}
	}

	supportedMessages := make(map[string]bool, len(supports.([]interface{})))
	for _, typeString := range supports.([]interface{}) {
		supportedMessages[typeString.(string)] = true
	}

	return &RemoteHandler{
		did:               handlerDid.(string),
		net:               net,
		supportedMessages: supportedMessages,
	}, nil
}
