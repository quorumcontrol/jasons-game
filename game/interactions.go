package game

import (
	"github.com/golang/protobuf/proto"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/typecaster"
)

func init() {
	cbor.RegisterCborType(RespondInteraction{})
	typecaster.AddType(RespondInteraction{})
	cbor.RegisterCborType(ChangeLocationInteraction{})
	typecaster.AddType(ChangeLocationInteraction{})
}

type Interaction interface {
	proto.Message
	GetCommand() string
}

var _ Interaction = (*RespondInteraction)(nil)
var _ Interaction = (*ChangeLocationInteraction)(nil)
