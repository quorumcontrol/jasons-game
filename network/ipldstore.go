package network

import (
	"context"

	lru "github.com/hashicorp/golang-lru"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	cbornode "github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

// This file is an experiment to see if we can use the IPLD
// block service on its own and also provide a TreeService to make
// getting/setting trees easier

type TreeStore interface {
	nodestore.DagStore

	GetTree(did string) (*consensus.SignedChainTree, error)
	SaveTreeMetadata(*consensus.SignedChainTree) error
	UpdateTreeMetadata(*consensus.SignedChainTree) error
}

type tipGetter interface {
	GetTip(did string) (cid.Cid, error)
}

type IPLDTreeStore struct {
	TreeStore
	blockApi    format.DAGService
	keyValueApi datastore.Batching
	tipGetter   tipGetter
	tipCache    *lru.Cache
}

func NewIPLDTreeStore(
	blockApi format.DAGService,
	keyValueApi datastore.Batching,
	tipGetter tipGetter,
) *IPLDTreeStore {
	cache, err := lru.New(100)
	if err != nil {
		panic("error creating LRU, should never happen")
	}

	return &IPLDTreeStore{
		blockApi:    blockApi,
		keyValueApi: keyValueApi,
		tipGetter:   tipGetter,
		tipCache:    cache,
	}
}

func (ts *IPLDTreeStore) getTip(did string) (tip cid.Cid, remote bool, err error) {
	ctx := context.TODO()

	tip, err = ts.getLocalTip(did)
	if err != nil {
		return tip, remote, errors.Wrap(err, "error getting local tip")
	}

	if !tip.Equals(cid.Undef) {
		log.Debug("found local tip", tip)
		return tip, remote, nil
	}

	// we didn't find it locally, let's go out and find it from the tipGetter (Tupelo)
	remote = true
	tip, err = ts.getRemoteTip(did)
	if err != nil {
		return tip, remote, errors.Wrap(err, "error getting remote tip")
	}

	log.Debug("found remote tip", tip)

	// ensure that remote signer didn't return us a stale tip
	// this can happen when blocks are played rapidly
	// and not all signers have processed the block
	cachedTip, found := ts.tipCache.Get(did)

	if found && tip.Equals(cid.Undef) {
		return cachedTip.(cid.Cid), remote, nil
	}

	if found && !cachedTip.(cid.Cid).Equals(tip) {
		log.Debug("found different cached tip", cachedTip.(cid.Cid))
		cachedTree := dag.NewDag(context.Background(), cachedTip.(cid.Cid), ts)
		cachedRoot := &chaintree.RootNode{}
		err = cachedTree.ResolveInto(ctx, []string{}, cachedRoot)
		if err != nil || cachedRoot == nil {
			log.Errorf("error checking cached height, continuing with remote tip")
			return tip, remote, nil
		}

		remoteTree := dag.NewDag(context.Background(), tip, ts)
		remoteRoot := &chaintree.RootNode{}
		err = remoteTree.ResolveInto(ctx, []string{}, remoteRoot)
		if err != nil || remoteRoot == nil {
			log.Errorf("error checking remote height, continuing with remote tip")
			return tip, remote, nil
		}
		// our local transactions have produced a more recent valid tip, so use it
		if cachedRoot.Height > remoteRoot.Height {
			return cachedTip.(cid.Cid), remote, nil
		}
	}

	return tip, remote, nil
}

func (ts *IPLDTreeStore) GetTree(did string) (*consensus.SignedChainTree, error) {
	ctx := context.TODO()

	log.Debug("IPLDTreeStore: GetTree did", did)

	tip, remote, err := ts.getTip(did)
	if err != nil {
		return nil, err
	}

	if tip.Equals(cid.Undef) {
		return nil, nil
	}

	log.Debugf("new dag")

	storedTree := dag.NewDag(ctx, tip, ts)
	log.Debugf("new tree")

	tree, err := chaintree.NewChainTree(ctx, storedTree, nil, consensus.DefaultTransactors)
	if err != nil {
		return nil, errors.Wrap(err, "error creating chaintree")
	}
	log.Debugf("get sigs")

	signedTree := &consensus.SignedChainTree{
		ChainTree: tree,
	}

	// TODO: support marshaling the remote signatures here
	if !remote {
		sigs, err := ts.getSignatures(did)
		if err != nil {
			return nil, errors.Wrap(err, "error getting signatures")
		}
		signedTree.Signatures = sigs
	}

	return signedTree, nil
}

func (ts *IPLDTreeStore) SaveTreeMetadata(tree *consensus.SignedChainTree) error {
	did, err := tree.Id()
	if err != nil {
		return errors.Wrap(err, "error getting id")
	}

	err = ts.setSignatures(did, tree.Signatures)
	if err != nil {
		return errors.Wrap(err, "error setting sigs")
	}

	ts.tipCache.Add(did, tree.Tip())
	return ts.keyValueApi.Put(didStoreKey(did), tree.Tip().Bytes())
}

func (ts *IPLDTreeStore) UpdateTreeMetadata(tree *consensus.SignedChainTree) error {
	did, err := tree.Id()
	if err != nil {
		return errors.Wrap(err, "error getting id")
	}

	log.Debug("UpdateTreeMetadata: Got chaintree DID: %s", did)

	has, err := ts.keyValueApi.Has(didStoreKey(did))
	if err != nil {
		return errors.Wrap(err, "error checking if tree metadata exists")
	}
	if has {
		log.Debug("UpdateTreeMetadata: Found chaintree in KV store")
		return ts.SaveTreeMetadata(tree)
	}

	log.Debug("UpdateTreeMetadata: Adding chaintree to tip cache")
	ts.tipCache.Add(did, tree.Tip())
	return nil
}

func (ts *IPLDTreeStore) Get(ctx context.Context, nodeCid cid.Cid) (format.Node, error) {
	log.Debug("IPLDTreeStore: Get node", nodeCid)
	return ts.blockApi.Get(ctx, nodeCid)
}

func (ts *IPLDTreeStore) GetMany(ctx context.Context, nodeCids []cid.Cid) <-chan *format.NodeOption {
	return ts.blockApi.GetMany(ctx, nodeCids)
}

func (ts *IPLDTreeStore) Add(ctx context.Context, node format.Node) error {
	err := ts.blockApi.Add(ctx, node)
	if err != nil {
		return errors.Wrap(err, "error adding blocks")
	}
	return err
}

func (ts *IPLDTreeStore) AddMany(ctx context.Context, nodes []format.Node) error {
	err := ts.blockApi.AddMany(ctx, nodes)
	if err != nil {
		return err
	}
	return nil
}

func (ts *IPLDTreeStore) Remove(ctx context.Context, nodeCid cid.Cid) error {
	return ts.blockApi.Remove(ctx, nodeCid)
}

func (ts *IPLDTreeStore) RemoveMany(ctx context.Context, nodeCids []cid.Cid) error {
	return ts.blockApi.RemoveMany(ctx, nodeCids)
}

func (ts *IPLDTreeStore) getLocalTip(did string) (cid.Cid, error) {
	log.Debugf("get local tip for %s", did)
	tipBytes, err := ts.keyValueApi.Get(didStoreKey(did))
	if err != nil {
		if err == datastore.ErrNotFound {
			return cid.Undef, nil
		}
		return cid.Undef, errors.Wrap(err, "error getting key")
	}
	tipCid, err := cid.Cast(tipBytes)
	if err != nil {
		return cid.Undef, errors.Wrap(err, "error casting tip")
	}
	return tipCid, nil
}

func (ts *IPLDTreeStore) getRemoteTip(did string) (cid.Cid, error) {
	log.Debugf("get remote tip for %s", did)
	tipCid, err := ts.tipGetter.GetTip(did)
	if err != nil {
		return cid.Undef, errors.Wrap(err, "error getting remote tip")
	}
	return tipCid, nil
}

func (ts *IPLDTreeStore) getSignatures(did string) (consensus.SignatureMap, error) {
	signatures, err := ts.keyValueApi.Get(didSignatureKey(did))
	if err != nil {
		return nil, errors.Wrap(err, "error getting signatures")
	}

	sigs := make(consensus.SignatureMap)
	if len(signatures) > 0 {
		err = cbornode.DecodeInto(signatures, &sigs)
		if err != nil {
			return nil, errors.Wrap(err, "error getting signatures")
		}
	}
	return sigs, nil
}

func (ts *IPLDTreeStore) setSignatures(did string, sigs consensus.SignatureMap) error {
	sw := safewrap.SafeWrap{}
	node := sw.WrapObject(sigs)
	if sw.Err != nil {
		return errors.Wrap(sw.Err, "error wrapping sigs")
	}
	return ts.keyValueApi.Put(didSignatureKey(did), node.RawData())
}

func didStoreKey(did string) datastore.Key {
	return datastore.NewKey(did)
}

func didSignatureKey(did string) datastore.Key {
	return datastore.NewKey("-s-" + did)
}
