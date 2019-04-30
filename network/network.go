package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-datastore"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/storage"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
)

type Network struct {
	Tupelo        *Tupelo
	Ipld          *ipfslite.Peer
	KeyValueStore datastore.Batching
}

func NewRemoteNetwork(ctx context.Context, group *types.NotaryGroup, path string) (*Network, error) {
	remote.Start()

	ds, err := ipfslite.BadgerDatastore(filepath.Join(path, "ipld"))
	if err != nil {
		return nil, fmt.Errorf("error creating store: %v", err)
	}
	net := &Network{
		KeyValueStore: ds,
	}

	key, err := net.GetOrCreatePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error getting private key: %v", err)
	}

	p2pHost, err := p2p.NewLibP2PHost(ctx, key, 0)
	if err != nil {
		return nil, fmt.Errorf("error setting up p2p host: %s", err)
	}
	if _, err = p2pHost.Bootstrap(p2p.BootstrapNodes()); err != nil {
		return nil, err
	}
	if err = p2pHost.WaitForBootstrap(len(group.Signers), 15*time.Second); err != nil {
		return nil, err
	}

	remote.NewRouter(p2pHost)
	group.SetupAllRemoteActors(&key.PublicKey)

	tupeloBadger, err := storage.NewBadgerStorage(filepath.Join(path, "tupelo"))
	if err != nil {
		return nil, fmt.Errorf("error creating storage: %v", err)
	}

	store := nodestore.NewStorageBasedStore(tupeloBadger)

	pubsub := remote.NewNetworkPubSub(p2pHost)

	tup := &Tupelo{
		key:          key,
		Store:        store,
		NotaryGroup:  group,
		PubSubSystem: pubsub,
	}
	net.Tupelo = tup

	lite, err := NewIPLDClient(ctx, ds)
	if err != nil {
		return nil, fmt.Errorf("error creating IPLD client: %v", err)
	}
	net.Ipld = lite

	return net, nil
}

func (n *Network) GetOrCreatePrivateKey() (*ecdsa.PrivateKey, error) {
	var key *ecdsa.PrivateKey

	storeKey := datastore.NewKey("privateKey")
	stored, err := n.KeyValueStore.Get(storeKey)
	if err == nil {
		reconstituted, err := crypto.ToECDSA(stored)
		if err != nil {
			return nil, fmt.Errorf("error putting key back together: %v", err)
		}
		key = reconstituted
	} else {
		if err != datastore.ErrNotFound {
			return nil, fmt.Errorf("error getting key: %v", err)
		}
		// key wasn't found generate a new key and save it
		newKey, err := crypto.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("error generating key: %v", err)
		}
		err = n.KeyValueStore.Put(storeKey, crypto.FromECDSA(newKey))
		if err != nil {
			return nil, fmt.Errorf("error putting key: %v", err)
		}
		key = newKey
	}

	return key, nil
}
