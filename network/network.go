package network

import (
	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-datastore"
)

type Network struct {
	Tupelo        *Tupelo
	Ipld          *ipfslite.Peer
	KeyValueStore datastore.Batching
}

func NewRemoteNetwork(path string) (*Network, error) {
	ds, err := ipfslite.BadgerDatastore(path)
	if err != nil {
		panic(err)
	}

	return &Network{
		
	}

}
