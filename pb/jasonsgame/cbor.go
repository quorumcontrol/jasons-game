package jasonsgame

import (
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/typecaster"
)

func init() {
	cbor.RegisterCborType(Location{})
	typecaster.AddType(Location{})
	cbor.RegisterCborType(Portal{})
	typecaster.AddType(Portal{})
}
