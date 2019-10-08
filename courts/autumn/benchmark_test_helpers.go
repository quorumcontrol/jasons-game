package autumn

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"

	messages "github.com/quorumcontrol/messages/build/go/community"

	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/handlers"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

type MockElementClient struct {
	Net      network.Network
	H        handlers.Handler
	Player   string
	Location string
}

func (e *MockElementClient) Send(id int) error {
	el, err := game.CreateObjectTree(e.Net, (&element{ID: id}).Name())
	if err != nil {
		return err
	}

	auths, err := e.serviceAuthentications()
	if err != nil {
		return err
	}

	_, err = e.Net.ChangeChainTreeOwner(el.ChainTree(), append(auths, crypto.PubkeyToAddress(*e.Net.PublicKey()).String()))
	if err != nil {
		return err
	}

	msg := &jasonsgame.TransferredObjectMessage{
		From:   e.Player,
		To:     e.Location,
		Object: el.MustId(),
	}
	return e.H.Handle(msg)
}

func (e *MockElementClient) PickupBowl() (*jasonsgame.TransferredObjectMessage, error) {
	playerInventory, err := trees.FindInventoryTree(e.Net, e.Player)
	if err != nil {
		return nil, err
	}
	received := make(chan *jasonsgame.TransferredObjectMessage, 1)
	sub, err := e.Net.Community().Subscribe(playerInventory.BroadcastTopic(), func(ctx context.Context, _ *messages.Envelope, msg proto.Message) {
		castMsg := msg.(*jasonsgame.TransferredObjectMessage)
		if castMsg != nil && castMsg.Object != "" && castMsg.Error == "" {
			_ = playerInventory.ForceAdd(castMsg.Object)
		}
		received <- castMsg
	})
	if err != nil {
		return nil, err
	}
	defer e.Net.Community().Unsubscribe(sub) // nolint

	msg := &jasonsgame.RequestObjectTransferMessage{
		From:   e.Location,
		To:     e.Player,
		Object: e.Bowl(),
	}

	_ = e.H.Handle(msg)

	select {
	case msg := <-received:
		return msg, nil
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("timeout waiting for transferred message")
	}
}

func (e *MockElementClient) serviceAuthentications() ([]string, error) {
	tree, err := e.Net.GetTree(e.Location)
	if err != nil {
		return nil, err
	}

	return tree.Authentications()
}

func (e *MockElementClient) Bowl() string {
	serviceInventory, err := trees.FindInventoryTree(e.Net, e.Location)
	if err != nil {
		return ""
	}
	bowlDid, _ := serviceInventory.DidForName(combinationObjectName)
	return bowlDid
}

func (e *MockElementClient) HasBowl() bool {
	return len(e.Bowl()) == 53
}

func (e *MockElementClient) HasElement(id int) bool {
	playerInventory, err := trees.FindInventoryTree(e.Net, e.Player)
	if err != nil {
		return false
	}

	elementDid, _ := playerInventory.DidForName((&element{ID: id}).Name())
	return len(elementDid) == 53
}
