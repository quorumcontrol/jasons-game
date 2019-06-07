package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

var log = logging.Logger("gamenetwork")

const BlockTopic = "jasons-game-tupelo-world-blocks"
const ShoutTopic = "jasons-game-shouting-players"
const GeneralTopic = "jasons-game-general"

var DefaultTupeloBootstrappers = []string{
	"/ip4/18.185.81.67/tcp/34001/ipfs/16Uiu2HAmJTmontYmNLgWPFLoWZYuEZ6fWhaqHh7vQABncaBWnZaW",
	"/ip4/3.217.212.32/tcp/34001/ipfs/16Uiu2HAmL5sbPp4LZJJhQTtTkpaNNEPUxrRoiyJqD8Mkj5tJkiow",
}

var DefaultGameBootstrappers = []string{
	"/ip4/3.208.36.214/tcp/34001/ipfs/16Uiu2HAmGsma99vu8SaheLdCEvMAH2VGbiQ1UH75ctjEVyz89ck6",
	"/ip4/13.57.66.151/tcp/34001/ipfs/16Uiu2HAmFsyL7pKNRYJAhsJCF9aMLajnr2DN8jskUx6bsVcumGhB",
}

type Network interface {
	Community() *Community
	ChangeChainTreeOwner(tree *consensus.SignedChainTree, newKeys []string) (*consensus.SignedChainTree, error)
	CreateChainTree() (*consensus.SignedChainTree, error)
	CreateNamedChainTree(name string) (*consensus.SignedChainTree, error)
	GetChainTreeByName(name string) (*consensus.SignedChainTree, error)
	GetTreeByTip(tip cid.Cid) (*consensus.SignedChainTree, error)
	GetTree(did string) (*consensus.SignedChainTree, error)
	UpdateChainTree(tree *consensus.SignedChainTree, path string, value interface{}) (*consensus.SignedChainTree, error)
	StartDiscovery(string) error
	StopDiscovery(string)
	WaitForDiscovery(ns string, num int, dur time.Duration) error
}

// RemoteNetwork implements the Network interface. Note this is *not* considered a secure system and private keys
// are stored on disk in plain text. It's "game-ready" security not "money-ready" security.
type RemoteNetwork struct {
	Tupelo        *Tupelo
	Ipld          *p2p.BitswapPeer
	KeyValueStore datastore.Batching
	TreeStore     TreeStore
	ipldp2pHost   *p2p.LibP2PHost
	community     *Community
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

	// TODO: keep the keys in a separate KeyStore
	key, err := net.getOrCreatePrivateKey()
	if err != nil {
		return nil, errors.Wrap(err, "error getting private key")
	}

	ipldNetHost, lite, err := NewIPLDClient(ctx, key, ds, p2p.WithClientOnlyDHT(true))
	if err != nil {
		return nil, errors.Wrap(err, "error creating IPLD client")
	}
	net.Ipld = lite
	net.ipldp2pHost = ipldNetHost
	net.community = NewJasonCommunity(ctx, key, ipldNetHost)
	pubSubSystem := remote.NewNetworkPubSub(ipldNetHost)

	// bootstrap to the game async so we can also setup the tupelo node, etc
	// while this happens.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := ipldNetHost.Bootstrap(GameBootstrappers())
		if err != nil {
			log.Errorf("error bootstrapping ipld host: %v", err)
			return
		}
		if err := pubSubSystem.Broadcast(BlockTopic, &Join{Identity: ipldNetHost.Identity()}); err != nil {
			log.Errorf("broadcasting Join failed: %s", err)
		}
	}()

	tupeloP2PHost, err := p2p.NewLibP2PHost(ctx, key, 0)
	if err != nil {
		return nil, fmt.Errorf("error setting up p2p host: %s", err)
	}

	remote.NewRouter(tupeloP2PHost)
	group.SetupAllRemoteActors(&key.PublicKey)

	tupeloPubSub := remote.NewNetworkPubSub(tupeloP2PHost)

	tup := &Tupelo{
		NotaryGroup:  group,
		PubSubSystem: tupeloPubSub,
	}
	net.Tupelo = tup

	store := NewIPLDTreeStore(lite, ds, pubSubSystem, tup)
	net.TreeStore = store
	tup.Store = store

	// now all that setup is done, wait for the tupelo and game bootstrappers

	if _, err = tupeloP2PHost.Bootstrap(TupeloBootstrappers()); err != nil {
		return nil, err
	}
	if err = tupeloP2PHost.WaitForBootstrap(len(group.Signers), 15*time.Second); err != nil {
		return nil, err
	}

	log.Infof("started tupelo host %s", tupeloP2PHost.Identity())
	wg.Wait() // wait for the game bootstrappers too
	log.Infof("connected to game bootstrappers")

	return net, nil
}

func (rn *RemoteNetwork) Community() *Community {
	return rn.community
}

func (rn *RemoteNetwork) StartDiscovery(ns string) error {
	return rn.ipldp2pHost.StartDiscovery(ns)
}

func (rn *RemoteNetwork) StopDiscovery(ns string) {
	rn.ipldp2pHost.StopDiscovery(ns)
}

func (rn *RemoteNetwork) WaitForDiscovery(ns string, num int, dur time.Duration) error {
	return rn.ipldp2pHost.WaitForDiscovery(ns, num, dur)
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

func (n *RemoteNetwork) mustPrivateKey() *ecdsa.PrivateKey {
	key, err := n.getOrCreatePrivateKey()
	if err != nil || key == nil {
		panic(errors.Wrap(err, "error getting or creating private key"))
	}
	return key
}

func (n *RemoteNetwork) CreateNamedChainTree(name string) (*consensus.SignedChainTree, error) {
	log.Debug("CreateNamedChainTree", name)
	tree, err := n.Tupelo.CreateChainTree(n.mustPrivateKey())
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

func (n *RemoteNetwork) CreateChainTree() (*consensus.SignedChainTree, error) {
	tree, err := n.Tupelo.CreateChainTree(n.mustPrivateKey())
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}
	log.Debug("CreateChainTree - created", tree.MustId())

	err = n.TreeStore.SaveTreeMetadata(tree)
	if err != nil {
		return nil, errors.Wrap(err, "error saving tree")
	}
	log.Debug("CreateChainTree - saved", tree.MustId())

	return tree, n.KeyValueStore.Put(datastore.NewKey("-n-"+tree.MustId()), []byte(tree.MustId()))
}

func (n *RemoteNetwork) GetChainTreeByName(name string) (*consensus.SignedChainTree, error) {
	log.Debugf("getchaintree by name")
	did, err := n.KeyValueStore.Get(datastore.NewKey("-n-" + name))
	if err == nil {
		return n.TreeStore.GetTree(string(did))
	}

	if len(did) == 0 || err == datastore.ErrNotFound {
		return nil, nil
	}
	return nil, errors.Wrap(err, "error getting tree")
}

func (n *RemoteNetwork) GetTree(did string) (*consensus.SignedChainTree, error) {
	return n.TreeStore.GetTree(did)
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
	err := n.Tupelo.UpdateChainTree(tree, n.mustPrivateKey(), path, value)
	if err != nil {
		return nil, errors.Wrap(err, "error updating chaintree")
	}
	return tree, n.TreeStore.SaveTreeMetadata(tree)
}

func (n *RemoteNetwork) ChangeChainTreeOwner(tree *consensus.SignedChainTree, newKeys []string) (*consensus.SignedChainTree, error) {
	log.Debug("ChangeChainTreeOwner", tree.MustId(), newKeys)

	transaction, err := chaintree.NewSetOwnershipTransaction(newKeys)
	if err != nil {
		return nil, errors.Wrap(err, "error updating chaintree")
	}

	err = n.Tupelo.PlayTransactions(tree, n.mustPrivateKey(), []*transactions.Transaction{transaction})
	if err != nil {
		return nil, errors.Wrap(err, "error updating chaintree")
	}
	return tree, n.TreeStore.SaveTreeMetadata(tree)
}

func TupeloBootstrappers() []string {
	if envSpecifiedNodes, ok := os.LookupEnv("TUPELO_BOOTSTRAP_NODES"); ok {
		log.Debugf("using tupelo bootstrap nodes: %s", envSpecifiedNodes)
		return strings.Split(envSpecifiedNodes, ",")
	}
	return DefaultTupeloBootstrappers
}

func GameBootstrappers() []string {
	if envSpecifiedNodes, ok := os.LookupEnv("JASON_BOOTSTRAP_NODES"); ok {
		log.Debugf("using jason bootstrap nodes: %s", envSpecifiedNodes)
		return strings.Split(envSpecifiedNodes, ",")
	}
	return DefaultGameBootstrappers
}
