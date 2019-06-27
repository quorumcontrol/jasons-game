package network

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
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
	key           *ecdsa.PrivateKey
	KeyValueStore datastore.Batching
	treeStore     TreeStore
	community     *Community
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
		key:           key,
		KeyValueStore: keystore,
		treeStore:     ipldstore,
		community:     NewJasonCommunity(context.Background(), key, ipldNetHost),
	}
}

func (ln *LocalNetwork) TreeStore() TreeStore {
	return ln.treeStore
}

func (ln *LocalNetwork) PublicKey() *ecdsa.PublicKey {
	return &ln.key.PublicKey
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

func (ln *LocalNetwork) CreateChainTree() (*consensus.SignedChainTree, error) {
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

	return tree, ln.KeyValueStore.Put(datastore.NewKey("-n-"+tree.MustId()), []byte(tree.MustId()))
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
	// TODO: if we enable this, we'll need to also do some sort of "insert" for test purposes
	return nil, fmt.Errorf("unimplemented")
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

type nilActor struct{}

func (_ *nilActor) Receive(_ actor.Context) {}

func (rn *LocalNetwork) NewCurrentStateSubscriptionProps(did string) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &nilActor{}
	})
}

func (ln *LocalNetwork) SendInk(tree *consensus.SignedChainTree, tokenName *consensus.TokenName, amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	// placeholder to fulfill the interface
	return nil, nil
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

	return tree, ln.TreeStore().SaveTreeMetadata(tree)
}
