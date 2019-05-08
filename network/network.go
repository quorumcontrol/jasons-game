package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	ipfslite "github.com/quorumcontrol/jasons-game/ipfslite"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

var log = logging.Logger("gamenetwork")

const KnownTopic = "brandon-topper-tupelo-game"

type Network interface {
	CreateNamedChainTree(name string) (*consensus.SignedChainTree, error)
	GetChainTreeByName(name string) (*consensus.SignedChainTree, error)
	GetRemoteTree(did string) (*consensus.SignedChainTree, error)
	GetTreeByTip(tip cid.Cid) (*consensus.SignedChainTree, error)
	UpdateChainTree(tree *consensus.SignedChainTree, path string, value interface{}) (*consensus.SignedChainTree, error)
	Publish(msg string)
}

type RemoteNetwork struct {
	Tupelo        *Tupelo
	Ipld          *ipfslite.Peer
	KeyValueStore datastore.Batching
	TreeStore     TreeStore
	PubSubSystem  remote.PubSub
}

func NewRemoteNetwork(ctx context.Context, group *types.NotaryGroup, path string) (Network, error) {
	remote.Start()

	ds, err := badger.NewDatastore(path, &badger.DefaultOptions)
	if err != nil {
		return nil, errors.Wrap(err, "error creating store")
	}
	net := &RemoteNetwork{
		KeyValueStore: ds,
	}

	lite, err := NewIPLDClient(ctx, ds)
	if err != nil {
		return nil, errors.Wrap(err, "error creating IPLD client")
	}
	net.Ipld = lite

	// TODO: keep the keys in a separate KeyStore
	key, err := net.getOrCreatePrivateKey()
	if err != nil {
		return nil, errors.Wrap(err, "error getting private key")
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

	store := NewIPLDTreeStore(lite, ds)
	net.TreeStore = store

	pubsub := remote.NewNetworkPubSub(p2pHost)
	net.PubSubSystem = pubsub

	subscriberFunc := func(aCtx actor.Context) {
		switch msg := aCtx.Message().(type) {
		case *messages.Ping:
			log.Infof("topic subscriber received ping: %s", msg.Msg)
		default:
			log.Debugf("received other message %v", msg)
		}
	}
	subscriber := actor.EmptyRootContext.Spawn(actor.PropsFromFunc(subscriberFunc))
	pubsub.Subscribe(actor.EmptyRootContext, KnownTopic, subscriber)

	tup := &Tupelo{
		key:          key,
		Store:        store,
		NotaryGroup:  group,
		PubSubSystem: pubsub,
	}
	net.Tupelo = tup

	return net, nil
}

func (n *RemoteNetwork) Publish(msg string) {
	n.PubSubSystem.Broadcast(KnownTopic, &messages.Ping{Msg: msg})
}

func (n *RemoteNetwork) getOrCreatePrivateKey() (*ecdsa.PrivateKey, error) {
	var key *ecdsa.PrivateKey

	storeKey := datastore.NewKey("privateKey")
	stored, err := n.KeyValueStore.Get(storeKey)
	if err == nil {
		reconstituted, err := crypto.ToECDSA(stored)
		if err != nil {
			return nil, errors.Wrap(err, "error putting key back together")
		}
		key = reconstituted
	} else {
		if err != datastore.ErrNotFound {
			return nil, errors.Wrap(err, "error getting key")
		}
		// key wasn't found generate a new key and save it
		newKey, err := crypto.GenerateKey()
		if err != nil {
			return nil, errors.Wrap(err, "error generating key")
		}
		err = n.KeyValueStore.Put(storeKey, crypto.FromECDSA(newKey))
		if err != nil {
			return nil, errors.Wrap(err, "error putting key")
		}
		key = newKey
	}

	return key, nil
}

func (n *RemoteNetwork) CreateNamedChainTree(name string) (*consensus.SignedChainTree, error) {
	log.Debug("CreateNamedChainTree", name)
	tree, err := n.Tupelo.CreateChainTree()
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}
	log.Debug("CreateNamedChainTree - created", name)

	err = n.TreeStore.SaveTreeMetadata(tree)
	if err != nil {
		return nil, errors.Wrap(err, "error saving tree")
	}
	log.Debug("CreateNamedChainTree - saved", name)

	return tree, n.KeyValueStore.Put(datastore.NewKey("-n-"+name), []byte(tree.MustId()))
}

func (n *RemoteNetwork) GetChainTreeByName(name string) (*consensus.SignedChainTree, error) {
	did, err := n.KeyValueStore.Get(datastore.NewKey("-n-" + name))
	if err == nil {
		return n.TreeStore.GetTree(string(did))
	}

	if len(did) == 0 || err == datastore.ErrNotFound {
		return nil, nil
	}
	return nil, errors.Wrap(err, "error getting tree")
}

func (n *RemoteNetwork) GetRemoteTree(did string) (*consensus.SignedChainTree, error) {
	tip, err := n.Tupelo.GetTip(did)
	if err != nil {
		return nil, errors.Wrap(err, "error getting tip")
	}
	return n.GetTreeByTip(tip)
}

func (n *RemoteNetwork) GetTreeByTip(tip cid.Cid) (*consensus.SignedChainTree, error) {
	storedTree := dag.NewDag(tip, n.TreeStore)

	tree, err := chaintree.NewChainTree(storedTree, nil, consensus.DefaultTransactors)
	if err != nil {
		return nil, errors.Wrap(err, "error creating chaintree")
	}

	return &consensus.SignedChainTree{
		ChainTree:  tree,
		Signatures: make(consensus.SignatureMap), // for now just ignore them
	}, nil
}

func (n *RemoteNetwork) UpdateChainTree(tree *consensus.SignedChainTree, path string, value interface{}) (*consensus.SignedChainTree, error) {
	log.Debug("updateTree", tree.MustId(), path, value)
	err := n.Tupelo.UpdateChainTree(tree, path, value)
	if err != nil {
		return nil, errors.Wrap(err, "error updating chaintree")
	}
	return tree, n.TreeStore.SaveTreeMetadata(tree)
}
