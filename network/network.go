package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	cid "github.com/ipfs/go-cid"
	datastore "github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/messages/build/go/signatures"
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

type InkNetwork interface {
	SendInk(tree *consensus.SignedChainTree, tokenName *consensus.TokenName, amount uint64, destinationChainId string) (*transactions.TokenPayload, error)
	ReceiveInk(tree *consensus.SignedChainTree, tokenPayload *transactions.TokenPayload) error
	ReceiveInkOnEphemeralChainTree(tree *consensus.SignedChainTree, privateKey *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) error
	DisallowReceiveInk(chaintreeId string)
}

type Network interface {
	InkNetwork
	Community() *Community
	ChangeChainTreeOwner(tree *consensus.SignedChainTree, newKeys []string) (*consensus.SignedChainTree, error)
	CreateChainTree() (*consensus.SignedChainTree, error)
	CreateEphemeralChainTree() (*consensus.SignedChainTree, *ecdsa.PrivateKey, error)
	CreateNamedChainTree(name string) (*consensus.SignedChainTree, error)
	GetChainTreeByName(name string) (*consensus.SignedChainTree, error)
	GetTreeByTip(tip cid.Cid) (*consensus.SignedChainTree, error)
	GetTree(did string) (*consensus.SignedChainTree, error)
	UpdateChainTree(tree *consensus.SignedChainTree, path string, value interface{}) (*consensus.SignedChainTree, error)
	TreeStore() TreeStore
	PublicKey() *ecdsa.PublicKey
	StartDiscovery(string) error
	StopDiscovery(string)
	WaitForDiscovery(ns string, num int, dur time.Duration) error
	NewCurrentStateSubscriptionProps(did string) *actor.Props
}

// RemoteNetwork implements the Network interface. Note this is *not* considered a secure system and private keys
// are stored on disk in plain text. It's "game-ready" security not "money-ready" security.
type RemoteNetwork struct {
	Tupelo        *Tupelo
	Ipld          *p2p.BitswapPeer
	KeyValueStore datastore.Batching
	treeStore     TreeStore
	ipldp2pHost   *p2p.LibP2PHost
	community     *Community
	signingKey    *ecdsa.PrivateKey
}

type RemoteNetworkConfig struct {
	NotaryGroup   *types.NotaryGroup
	KeyValueStore datastore.Batching
	SigningKey    *ecdsa.PrivateKey
	NetworkKey    *ecdsa.PrivateKey
	IpldKey       *ecdsa.PrivateKey
}

var _ Network = &RemoteNetwork{}

func NewRemoteNetworkWithConfig(ctx context.Context, config *RemoteNetworkConfig) (Network, error) {
	var err error

	remote.Start()

	net := &RemoteNetwork{
		KeyValueStore: config.KeyValueStore,
		signingKey:    config.SigningKey,
	}
	group := config.NotaryGroup

	networkKey := config.NetworkKey
	if networkKey == nil {
		networkKey, err = crypto.GenerateKey()
		if err != nil {
			return nil, errors.Wrap(err, "error generating network key")
		}
	}

	ipldKey := config.IpldKey
	if ipldKey == nil {
		ipldKey, err = crypto.GenerateKey()
		if err != nil {
			return nil, errors.Wrap(err, "error generating ipld key")
		}
	}

	ipldNetHost, lite, err := NewIPLDClient(ctx, ipldKey, net.KeyValueStore, p2p.WithClientOnlyDHT(true))
	if err != nil {
		return nil, errors.Wrap(err, "error creating IPLD client")
	}
	net.Ipld = lite
	net.ipldp2pHost = ipldNetHost
	net.community = NewJasonCommunity(ctx, ipldKey, ipldNetHost)

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
		if err := net.WaitForDiscovery(DiscoveryNamespace, 1, 10*time.Second); err != nil {
			log.Errorf("waiting for discovery failed %s", err)
			return
		}
		if err := net.community.Send([]byte(GeneralTopic), &Join{Identity: ipldNetHost.Identity()}); err != nil {
			log.Errorf("broadcasting Join failed: %s", err)
		}
	}()

	tupeloP2PHost, err := p2p.NewLibP2PHost(ctx, networkKey, 0)
	if err != nil {
		return nil, fmt.Errorf("error setting up p2p host: %s", err)
	}

	remote.NewRouter(tupeloP2PHost)
	group.SetupAllRemoteActors(&networkKey.PublicKey)

	tupeloPubSub := remote.NewNetworkPubSub(tupeloP2PHost.GetPubSub())

	tup := &Tupelo{
		NotaryGroup:  group,
		PubSubSystem: tupeloPubSub,
	}
	net.Tupelo = tup

	store := NewIPLDTreeStore(lite, net.KeyValueStore, tup)
	net.treeStore = store
	tup.Store = store

	// now all that setup is done, wait for the tupelo and game bootstrappers

	if _, err = tupeloP2PHost.Bootstrap(group.Config().BootstrapAddresses); err != nil {
		return nil, errors.Wrap(err, "error bootstrapping to tupelo")
	}
	if err = tupeloP2PHost.WaitForBootstrap(len(group.Signers), 15*time.Second); err != nil {
		return nil, errors.Wrap(err, "error on bootstrap wait for tupelo")
	}

	log.Infof("started tupelo host %s", tupeloP2PHost.Identity())
	wg.Wait() // wait for the game bootstrappers too
	log.Infof("connected to game bootstrappers")

	return net, nil
}

func NewRemoteNetwork(ctx context.Context, group *types.NotaryGroup, ds datastore.Batching) (Network, error) {
	// TODO: keep the keys in a separate KeyStore
	key, err := GetOrCreateStoredPrivateKey(ds)
	if err != nil {
		return nil, errors.Wrap(err, "error getting private key")
	}

	return NewRemoteNetworkWithConfig(ctx, &RemoteNetworkConfig{
		NotaryGroup:   group,
		SigningKey:    key,
		NetworkKey:    key,
		KeyValueStore: ds,
	})
}

func (rn *RemoteNetwork) TreeStore() TreeStore {
	return rn.treeStore
}

func (rn *RemoteNetwork) PublicKey() *ecdsa.PublicKey {
	return &rn.PrivateKey().PublicKey
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

func GetOrCreateStoredPrivateKey(ds datastore.Batching) (key *ecdsa.PrivateKey, err error) {
	storeKey := datastore.NewKey("privateKey")
	stored, err := ds.Get(storeKey)
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
		err = ds.Put(storeKey, crypto.FromECDSA(newKey))
		if err != nil {
			return nil, errors.Wrap(err, "error putting key")
		}
		key = newKey
	}

	return key, nil
}

func (n *RemoteNetwork) PrivateKey() *ecdsa.PrivateKey {
	return n.signingKey
}

func (n *RemoteNetwork) CreateNamedChainTree(name string) (*consensus.SignedChainTree, error) {
	log.Debug("CreateNamedChainTree", name)
	tree, err := n.Tupelo.CreateChainTree(n.PrivateKey())
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}
	log.Debug("CreateNamedChainTree - created", name)

	err = n.TreeStore().SaveTreeMetadata(tree)
	if err != nil {
		return nil, errors.Wrap(err, "error saving tree")
	}
	log.Debug("CreateNamedChainTree - saved", name)

	return tree, n.KeyValueStore.Put(datastore.NewKey("-n-"+name), []byte(tree.MustId()))
}

func (n *RemoteNetwork) createChainTree(key *ecdsa.PrivateKey) (*consensus.SignedChainTree, error) {
	tree, err := n.Tupelo.CreateChainTree(key)
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}
	log.Debug("CreateChainTree - created", tree.MustId())

	err = n.TreeStore().SaveTreeMetadata(tree)
	if err != nil {
		return nil, errors.Wrap(err, "error saving tree")
	}
	log.Debug("CreateChainTree - saved", tree.MustId())

	return tree, n.KeyValueStore.Put(datastore.NewKey("-n-"+tree.MustId()), []byte(tree.MustId()))
}

func (n *RemoteNetwork) CreateChainTree() (*consensus.SignedChainTree, error) {
	return n.createChainTree(n.PrivateKey())
}

func (n *RemoteNetwork) CreateEphemeralChainTree() (*consensus.SignedChainTree, *ecdsa.PrivateKey, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "error generating ephemeral private key")
	}

	ct, err := n.createChainTree(key)

	return ct, key, err
}

func (n *RemoteNetwork) GetChainTreeByName(name string) (*consensus.SignedChainTree, error) {
	log.Debugf("getchaintree by name")
	did, err := n.KeyValueStore.Get(datastore.NewKey("-n-" + name))
	if err == nil {
		return n.TreeStore().GetTree(string(did))
	}

	if len(did) == 0 || err == datastore.ErrNotFound {
		return nil, nil
	}
	return nil, errors.Wrap(err, "error getting tree")
}

func (n *RemoteNetwork) GetTree(did string) (*consensus.SignedChainTree, error) {
	return n.TreeStore().GetTree(did)
}

func (n *RemoteNetwork) GetTreeByTip(tip cid.Cid) (*consensus.SignedChainTree, error) {
	ctx := context.TODO()
	storedTree := dag.NewDag(ctx, tip, n.TreeStore())

	tree, err := chaintree.NewChainTree(ctx, storedTree, nil, consensus.DefaultTransactors)
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
	err := n.Tupelo.UpdateChainTree(tree, n.PrivateKey(), path, value)
	if err != nil {
		return nil, errors.Wrap(err, "error updating chaintree")
	}
	return tree, n.TreeStore().SaveTreeMetadata(tree)
}

func (n *RemoteNetwork) ChangeChainTreeOwner(tree *consensus.SignedChainTree, newKeys []string) (*consensus.SignedChainTree, error) {
	log.Debug("ChangeChainTreeOwner", tree.MustId(), newKeys)

	transaction, err := chaintree.NewSetOwnershipTransaction(newKeys)
	if err != nil {
		return nil, errors.Wrap(err, "error updating chaintree")
	}

	_, err = n.Tupelo.PlayTransactions(tree, n.PrivateKey(), []*transactions.Transaction{transaction})
	if err != nil {
		return nil, errors.Wrap(err, "error updating chaintree")
	}
	return tree, n.TreeStore().SaveTreeMetadata(tree)
}

type currentStateSubscriptionActor struct {
	did    string
	tupelo *Tupelo
	cancel func()
}

func (act *currentStateSubscriptionActor) Receive(actorContext actor.Context) {
	switch actorContext.Message().(type) {
	case *actor.Started:
		var err error
		act.cancel, err = act.tupelo.SubscribeToCurrentStateChanges(act.did, func(msg *signatures.CurrentState) {
			actorContext.Send(actorContext.Parent(), msg)
		})
		if err != nil {
			panic(errors.Wrap(err, "error starting subscription actor for current state"))
		}
	case *actor.Stopping:
		act.cancel()
	}
}

func (rn *RemoteNetwork) NewCurrentStateSubscriptionProps(did string) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &currentStateSubscriptionActor{
			did:    did,
			tupelo: rn.Tupelo,
		}
	})
}

func (n *RemoteNetwork) SendInk(tree *consensus.SignedChainTree, tokenName *consensus.TokenName, amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	transactionId, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "error generating send token transaction ID")
	}

	log.Debugf("send ink transaction id: %s", transactionId)

	transaction, err := chaintree.NewSendTokenTransaction(transactionId.String(), tokenName.String(), amount, destinationChainId)
	if err != nil {
		return nil, errors.Wrap(err, "error generating ink send token transaction")
	}

	log.Debugf("send ink transaction: %+v", *transaction)

	txResp, err := n.Tupelo.PlayTransactions(tree, n.PrivateKey(), []*transactions.Transaction{transaction})
	if err != nil {
		return nil, errors.Wrap(err, "error playing ink send token transaction")
	}

	log.Debugf("send ink PlayTransactions response: %+v", txResp)

	err = n.TreeStore().SaveTreeMetadata(tree)
	if err != nil {
		return nil, errors.Wrap(err, "error saving chaintree metadata after ink send transaction")
	}

	log.Debug("send ink saved tree metadata")

	tokenPayload, err := n.Tupelo.TokenPayloadForTransaction(tree, tokenName, transactionId.String(), &txResp.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "error getting token payload for ink send")
	}

	log.Debugf("send ink token payload: %+v", *tokenPayload)

	return tokenPayload, nil
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
