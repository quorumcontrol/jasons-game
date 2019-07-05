package network

import (
	"context"
	"strings"
	"time"

	dshelp "github.com/ipfs/go-ipfs-ds-help"

	"github.com/ipfs/go-datastore/query"

	"github.com/AsynkronIT/protoactor-go/actor"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	cbornode "github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
)

// This file is an experiment to see if we can use the IPLD
// block service on its own and also provide a TreeService to make
// getting/setting trees easier

type TreeStore interface {
	nodestore.DagStore

	GetTree(did string) (*consensus.SignedChainTree, error)
	SaveTreeMetadata(*consensus.SignedChainTree) error
}

type tipGetter interface {
	GetTip(did string) (cid.Cid, error)
}

type IPLDTreeStore struct {
	TreeStore
	publisher   *actor.PID
	blockApi    format.DAGService
	keyValueApi datastore.Batching
	tipGetter   tipGetter
}

func NewIPLDTreeStore(
	blockApi format.DAGService,
	keyValueApi datastore.Batching,
	pubsubSystem remote.PubSub,
	tipGetter tipGetter,
) *IPLDTreeStore {
	return &IPLDTreeStore{
		blockApi:    blockApi,
		keyValueApi: keyValueApi,
		publisher:   actor.EmptyRootContext.Spawn(newPublisherProps(pubsubSystem)),
		tipGetter:   tipGetter,
	}
}

func (ts *IPLDTreeStore) GetTree(did string) (*consensus.SignedChainTree, error) {
	ctx := context.TODO()
	log.Debugf("get local tip")
	var remote bool
	tip, err := ts.getLocalTip(did)
	if err != nil {
		return nil, errors.Wrap(err, "error getting local tip")
	}

	if tip.Equals(cid.Undef) {
		remote = true
		// if we didn't find it locally, let's go out and find it from the tipGetter (Tupelo)
		tip, err = ts.getRemoteTip(did)
		if err != nil {
			return nil, errors.Wrap(err, "error getting remote tip")
		}
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
	return ts.keyValueApi.Put(didStoreKey(did), tree.Tip().Bytes())
}

func (ts *IPLDTreeStore) Get(ctx context.Context, nodeCid cid.Cid) (format.Node, error) {
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
	if ts.publisher != nil {
		actor.EmptyRootContext.Send(ts.publisher, node)
	}
	return err
}

func (ts *IPLDTreeStore) AddMany(ctx context.Context, nodes []format.Node) error {
	err := ts.blockApi.AddMany(ctx, nodes)
	if err != nil {
		return err
	}

	if ts.publisher != nil {
		for _, n := range nodes {
			actor.EmptyRootContext.Send(ts.publisher, n)
		}
	}
	return nil
}

func (ts *IPLDTreeStore) Remove(ctx context.Context, nodeCid cid.Cid) error {
	return ts.blockApi.Remove(ctx, nodeCid)
}

func (ts *IPLDTreeStore) RemoveMany(ctx context.Context, nodeCids []cid.Cid) error {
	return ts.blockApi.RemoveMany(ctx, nodeCids)
}

func (ts *IPLDTreeStore) RepublishAll() error {
	result, err := ts.keyValueApi.Query(query.Query{
		Prefix:   blockstore.BlockPrefix.String(),
		KeysOnly: true,
	})
	if err != nil {
		return errors.Wrap(err, "error querying")
	}
	for entry := range result.Next() {
		keyStr := entry.Key
		key := datastore.NewKey(strings.Split(keyStr, "/")[2])

		cid, err := dshelp.DsKeyToCid(key)
		if err != nil {
			return errors.Wrap(err, "error getting cid")
		}

		node, err := ts.Get(context.TODO(), cid)
		if err != nil {
			return errors.Wrap(err, "error getting CID")
		}
		log.Infof("publishing %s", node.Cid().String())
		actor.EmptyRootContext.Send(ts.publisher, node)
		time.Sleep(15 * time.Millisecond)
	}
	return nil
}

func (ts *IPLDTreeStore) getLocalTip(did string) (cid.Cid, error) {
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
