package inventory

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/protobuf/proto"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/services/handlers"
)

var log = logging.Logger("inventory-handler")

type InvetoryHandler struct {
	network network.Network
}

func NewHandlerProps(network network.Network) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &InvetoryHandler{
			network: network,
		}
	})
}

func (h *InvetoryHandler) SupportedMessages() []string {
	return []string{
		proto.MessageName(&jasonsgame.TransferredObjectMessage{}),
	}
}

func (h *InvetoryHandler) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *handlers.GetSupportedMessages:
		actorCtx.Respond(h.SupportedMessages())
	case *jasonsgame.TransferredObjectMessage:
		inventoryActor := actorCtx.Spawn(game.NewInventoryActorProps(&game.InventoryActorConfig{
			Did:     msg.GetTo(),
			Network: h.network,
		}))

		actorCtx.Send(inventoryActor, msg)

		err := actorCtx.PoisonFuture(inventoryActor).Wait()
		if err != nil {
			log.Errorf("error stopping inventory actor, %v", err)
		}
	}
}
