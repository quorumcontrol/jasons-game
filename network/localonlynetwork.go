package network

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipfs/go-blockservice"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/ipfs/go-merkledag"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

type DevNullTipGetter struct{}

func (dntg *DevNullTipGetter) GetTip(_ string) (cid.Cid, error) {
	return cid.Undef, nil
}

// LocalNetwork implements the Network interface but doesn't require
// a full tupelo/IPLD setup
type LocalNetwork struct {
	key              *ecdsa.PrivateKey
	KeyValueStore    datastore.Batching
	treeStore        TreeStore
	community        *Community
	mockTupeloEvents *eventstream.EventStream
}

var _ Network = &LocalNetwork{}

func NewLocalNetwork() *LocalNetwork {
	keystore := dssync.MutexWrap(datastore.NewMapDatastore())

	bstore := blockstore.NewBlockstore(keystore)
	bserv := blockservice.New(bstore, offline.Exchange(bstore))
	dag := merkledag.NewDAGService(bserv)
	ds := datastore.NewMapDatastore()

	key, err := crypto.GenerateKey()
	if err != nil {
		panic(errors.Wrap(err, "error generating key"))
	}

	ipldNetHost, _, err := NewIPLDClient(context.Background(), key, ds, p2p.WithClientOnlyDHT(true))
	if err != nil {
		panic(errors.Wrap(err, "error creating IPLD client"))
	}

	ipldstore := NewIPLDTreeStore(dag, keystore, new(DevNullTipGetter))

	return &LocalNetwork{
		key:              key,
		KeyValueStore:    keystore,
		treeStore:        ipldstore,
		community:        NewJasonCommunity(context.Background(), key, ipldNetHost),
		mockTupeloEvents: new(eventstream.EventStream),
	}
}

func (ln *LocalNetwork) TreeStore() TreeStore {
	return ln.treeStore
}

func (ln *LocalNetwork) PublicKey() *ecdsa.PublicKey {
	return &ln.key.PublicKey
}

func (ln *LocalNetwork) PrivateKey() *ecdsa.PrivateKey {
	return ln.key
}

func (ln *LocalNetwork) Community() *Community {
	return ln.community
}

func (ln *LocalNetwork) StartDiscovery(_ string) error {
	//noop
	return nil
}

func (ln *LocalNetwork) StopDiscovery(_ string) {
	//noop
}

func (ln *LocalNetwork) WaitForDiscovery(ns string, num int, dur time.Duration) error {
	//noop
	return nil
}

func (ln *LocalNetwork) CreateLocalChainTree(name string) (*consensus.SignedChainTree, error) {
	return ln.CreateNamedChainTree(name)
}

// in real network, passphrase tree is stateless - but for localnetwork, we can assume its a local
// tree, so just use the passphrase for a "named chaintree"
func (ln *LocalNetwork) FindOrCreatePassphraseTree(passphrase string) (*consensus.SignedChainTree, error) {
	seed := sha256.Sum256([]byte(passphrase))
	name := string(seed[:32])

	tree, err := ln.GetChainTreeByName(name)
	if err != nil {
		return nil, errors.Wrap(err, "getting passphrase chaintree")
	}

	if tree == nil {
		return ln.CreateNamedChainTree(name)
	}
	return tree, nil
}

func (ln *LocalNetwork) CreateNamedChainTree(name string) (*consensus.SignedChainTree, error) {
	ephemeralPrivate, err := crypto.GenerateKey()
	if err != nil {
		return nil, errors.Wrap(err, "error creating key")
	}

	tree, err := consensus.NewSignedChainTree(ephemeralPrivate.PublicKey, ln.TreeStore())
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}

	newTree, err := ln.ChangeChainTreeOwner(tree, []string{crypto.PubkeyToAddress(ln.key.PublicKey).String()})
	if err != nil {
		return nil, errors.Wrap(err, "error changing ownership")
	}
	tree = newTree

	err = ln.TreeStore().SaveTreeMetadata(tree)
	if err != nil {
		return nil, errors.Wrap(err, "error saving tree metadata")
	}

	return tree, ln.KeyValueStore.Put(datastore.NewKey("-n-"+name), []byte(tree.MustId()))
}

func (ln *LocalNetwork) CreateChainTreeWithKey(key *ecdsa.PrivateKey) (*consensus.SignedChainTree, error) {
	tree, err := consensus.NewSignedChainTree(key.PublicKey, ln.TreeStore())
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}

	newTree, err := ln.ChangeChainTreeOwner(tree, []string{crypto.PubkeyToAddress(ln.key.PublicKey).String()})
	if err != nil {
		return nil, errors.Wrap(err, "error changing ownership")
	}
	tree = newTree

	err = ln.TreeStore().SaveTreeMetadata(tree)
	if err != nil {
		return nil, errors.Wrap(err, "error saving tree metadata")
	}

	return tree, ln.KeyValueStore.Put(datastore.NewKey("-n-"+tree.MustId()), []byte(tree.MustId()))
}

func (ln *LocalNetwork) CreateChainTree() (*consensus.SignedChainTree, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	ct, err := ln.CreateChainTreeWithKey(key)
	return ct, err
}

func (ln *LocalNetwork) GetChainTreeByName(name string) (*consensus.SignedChainTree, error) {
	did, err := ln.KeyValueStore.Get(datastore.NewKey("-n-" + name))
	if err == nil {
		return ln.TreeStore().GetTree(string(did))
	}

	if err == datastore.ErrNotFound {
		return nil, nil
	}
	return nil, errors.Wrap(err, "error getting tree")
}

func (ln *LocalNetwork) GetTree(did string) (*consensus.SignedChainTree, error) {
	return ln.TreeStore().GetTree(did)
}

func (ln *LocalNetwork) GetTreeByTip(tip cid.Cid) (*consensus.SignedChainTree, error) {
	rootNode, err := ln.TreeStore().Get(context.Background(), tip)
	if err != nil {
		return nil, err
	}

	didUncast, _, err := rootNode.Resolve([]string{"id"})
	if err != nil {
		return nil, err
	}

	if didUncast == nil {
		return nil, nil
	}

	return ln.GetTree(didUncast.(string))
}

func (ln *LocalNetwork) UpdateChainTree(tree *consensus.SignedChainTree, path string, value interface{}) (*consensus.SignedChainTree, error) {
	transaction, err := chaintree.NewSetDataTransaction(path, value)
	if err != nil {
		return nil, err
	}
	return ln.playTransactions(tree, []*transactions.Transaction{transaction})
}

func (ln *LocalNetwork) ChangeChainTreeOwner(tree *consensus.SignedChainTree, newKeys []string) (*consensus.SignedChainTree, error) {
	transaction, err := chaintree.NewSetOwnershipTransaction(newKeys)
	if err != nil {
		return nil, err
	}
	return ln.playTransactions(tree, []*transactions.Transaction{transaction})
}

func (ln *LocalNetwork) ChangeChainTreeOwnerWithKey(tree *consensus.SignedChainTree, privateKey *ecdsa.PrivateKey, newKeys []string) (*consensus.SignedChainTree, error) {
	// placeholder to fulfill the interface
	return nil, nil
}

func (ln *LocalNetwork) DeleteTree(name string) error {
	// placeholder to fulfill the interface
	return nil
}

func (rn *LocalNetwork) NewCurrentStateSubscriptionProps(did string) *actor.Props {
	return actor.PropsFromFunc(func(actorCtx actor.Context) {
		switch actorCtx.Message().(type) {
		case *actor.Started:
			rn.mockTupeloEvents.Subscribe(func(evt interface{}) {
				switch eMsg := evt.(type) {
				case *signatures.CurrentState:
					if did == string(eMsg.Signature.ObjectId) {
						actorCtx.Send(actorCtx.Parent(), evt)
					}
				}
			})
		}
	})
}

func (ln *LocalNetwork) SendInk(tree *consensus.SignedChainTree, tokenName *consensus.TokenName, amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	// placeholder to fulfill the interface
	return nil, nil
}

func (ln *LocalNetwork) ReceiveInk(tree *consensus.SignedChainTree, tokenPayload *transactions.TokenPayload) error {
	// placeholder to fulfill the interface
	return nil
}

func (ln *LocalNetwork) ReceiveInkOnEphemeralChainTree(tree *consensus.SignedChainTree, privateKey *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) error {
	// placeholder to fulfill the interface
	return nil
}

func (ln *LocalNetwork) DisallowReceiveInk(chaintreeId string) {
	// placeholder to fulfill the interface
}

func (ln *LocalNetwork) playTransactions(tree *consensus.SignedChainTree, transactions []*transactions.Transaction) (*consensus.SignedChainTree, error) {
	ctx := context.TODO()
	unmarshaledRoot, err := tree.ChainTree.Dag.Get(ctx, tree.Tip())
	if unmarshaledRoot == nil || err != nil {
		return nil, fmt.Errorf("error missing root: %v", err)
	}
	root := &chaintree.RootNode{}

	err = cbornode.DecodeInto(unmarshaledRoot.RawData(), root)
	if err != nil {
		return nil, fmt.Errorf("error decoding root: %v", err)
	}

	var height uint64
	var tip *cid.Cid
	if tree.IsGenesis() {
		height = 0
	} else {
		height = root.Height + 1
		storedTip := tree.Tip()
		tip = &storedTip
	}

	unsignedBlock := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  tip,
			Height:       height,
			Transactions: transactions,
		},
	}

	blockWithHeaders, err := consensus.SignBlock(unsignedBlock, ln.key)
	if err != nil {
		return nil, fmt.Errorf("error signing root: %v", err)
	}

	isValid, err := tree.ChainTree.ProcessBlock(ctx, blockWithHeaders)
	if err != nil {
		return nil, fmt.Errorf("error processing block: %v", err)
	}

	if !isValid {
		return nil, fmt.Errorf("error invalid transaction")
	}

	currentState := &signatures.CurrentState{Signature: &signatures.Signature{
		ObjectId: []byte(tree.MustId()),
		NewTip:   tree.Tip().Bytes(),
		Height:   height + 1,
	}}
	if tip != nil {
		currentState.Signature.PreviousTip = tip.Bytes()
	}
	ln.mockTupeloEvents.Publish(currentState)

	return tree, ln.TreeStore().SaveTreeMetadata(tree)
}
