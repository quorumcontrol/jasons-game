package network

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipfs/go-blockservice"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	"github.com/ipfs/go-merkledag"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
)

// LocalNetwork implements the Network interface but doesn't require
// a full tupelo/IPLD setup
type LocalNetwork struct {
	key           *ecdsa.PrivateKey
	KeyValueStore datastore.Batching
	TreeStore     TreeStore
}

func NewLocalNetwork() Network {
	keystore := datastore.NewMapDatastore()

	bstore := blockstore.NewBlockstore(keystore)
	bserv := blockservice.New(bstore, offline.Exchange(bstore))
	dag := merkledag.NewDAGService(bserv)

	ipldstore := NewIPLDTreeStore(dag, keystore)

	key, err := crypto.GenerateKey()
	if err != nil {
		panic(errors.Wrap(err, "error generating key"))
	}

	return &LocalNetwork{
		key:           key,
		KeyValueStore: keystore,
		TreeStore:     ipldstore,
	}
}

func (ln *LocalNetwork) CreateNamedChainTree(name string) (*consensus.SignedChainTree, error) {
	ephemeralPrivate, err := crypto.GenerateKey()
	if err != nil {
		return nil, errors.Wrap(err, "error creating key")
	}

	tree, err := consensus.NewSignedChainTree(ephemeralPrivate.PublicKey, ln.TreeStore)
	if err != nil {
		return nil, errors.Wrap(err, "error creating tree")
	}

	// TODO: do we need to actually set the authorizations here? or do we just not care

	err = ln.TreeStore.SaveTreeMetadata(tree)
	if err != nil {
		return nil, errors.Wrap(err, "error saving tree metadata")
	}

	return tree, ln.KeyValueStore.Put(datastore.NewKey("-n-"+name), []byte(tree.MustId()))
}

func (ln *LocalNetwork) GetChainTreeByName(name string) (*consensus.SignedChainTree, error) {
	did, err := ln.KeyValueStore.Get(datastore.NewKey("-n-" + name))
	if err == nil {
		return ln.TreeStore.GetTree(string(did))
	}

	if err == datastore.ErrNotFound {
		return nil, nil
	}
	return nil, errors.Wrap(err, "error getting tree")
}

func (ln *LocalNetwork) GetRemoteTree(did string) (*consensus.SignedChainTree, error) {
	// TODO: if we enable this, we'll need to also do some sort of "insert" for test purposes
	return nil, fmt.Errorf("unimplemented")
}

func (ln *LocalNetwork) GetTreeByTip(tip cid.Cid) (*consensus.SignedChainTree, error) {
	// TODO: if we enable this, we'll need to also do some sort of "insert" for test purposes
	return nil, fmt.Errorf("unimplemented")
}

func (ln *LocalNetwork) UpdateChainTree(tree *consensus.SignedChainTree, path string, value interface{}) (*consensus.SignedChainTree, error) {
	updated, err := tree.ChainTree.Dag.SetAsLink(append([]string{"tree", "data"}, strings.Split(path, "/")...), value)
	if err != nil {
		return nil, errors.Wrap(err, "error setting data")
	}
	tree.ChainTree.Dag = updated

	return tree, ln.TreeStore.SaveTreeMetadata(tree)
}
