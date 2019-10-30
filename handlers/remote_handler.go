package handlers

import (
	"context"
	"crypto/ecdsa"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

type RemoteHandler struct {
	Handler
	did               string
	net               network.Network
	supportedMessages HandlerMessageList
}

func (h *RemoteHandler) Handle(msg proto.Message) error {
	if !h.Supports(msg) {
		return ErrUnsupportedMessageType
	}
	topic := h.net.Community().TopicFor(h.did)
	return h.net.Community().Send(topic, msg)
}

func (h *RemoteHandler) Supports(msg proto.Message) bool {
	return h.supportedMessages.Contains(msg)
}

func (h *RemoteHandler) SupportedMessages() []string {
	return h.supportedMessages
}

func (h *RemoteHandler) Did() string {
	return h.did
}

func (h *RemoteHandler) PeerPublicKeys() ([]*ecdsa.PublicKey, error) {
	ctx := context.Background()

	tree, err := h.net.GetTree(h.Did())

	if err != nil {
		return nil, errors.Wrap(err, "error fetching handler tree")
	}

	peersUncast, _, err := tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "handler", "peers"})
	if err != nil {
		return nil, errors.Wrap(err, "error fetching handler tree")
	}

	if peersUncast == nil {
		return nil, nil
	}

	peerPubKeys := make([]*ecdsa.PublicKey, len(peersUncast.([]interface{})))
	for i, peerIDStr := range peersUncast.([]interface{}) {
		peerID, err := peer.IDB58Decode(peerIDStr.(string))
		if err != nil {
			return nil, errors.Wrap(err, "error decoding peer id")
		}

		peerPubKeys[i], err = p2p.EcdsaKeyFromPeer(peerID)
		if err != nil {
			return nil, errors.Wrap(err, "error getting handler peer pubkeys")
		}
	}

	return peerPubKeys, nil
}

func FindHandlerForTree(net network.Network, did string) (*RemoteHandler, error) {
	ctx := context.TODO()

	tree, err := net.GetTree(did)
	if err != nil {
		return nil, err
	}

	handlerDid, _, err := tree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", HandlerPath})
	if err != nil {
		return nil, err
	}

	if handlerDid == nil {
		return nil, nil
	}

	return GetRemoteHandler(net, handlerDid.(string))
}

func GetRemoteHandler(net network.Network, handlerDid string) (*RemoteHandler, error) {
	ctx := context.TODO()

	handlerTree, err := net.GetTree(handlerDid)
	if err != nil {
		return nil, err
	}

	supports, _, err := handlerTree.ChainTree.Dag.Resolve(ctx, []string{"tree", "data", "jasons-game", "handler", "supports"})
	if err != nil {
		return nil, err
	}
	if supports == nil {
		supports = []interface{}{}
	}

	supportedMessages := make(HandlerMessageList, len(supports.([]interface{})))
	for i, typeString := range supports.([]interface{}) {
		supportedMessages[i] = typeString.(string)
	}

	return &RemoteHandler{
		did:               handlerDid,
		net:               net,
		supportedMessages: supportedMessages,
	}, nil
}
