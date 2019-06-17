package network

import (
	"context"
	"fmt"
	"strings"
	"time"

	dshelp "github.com/ipfs/go-ipfs-ds-help"

	"github.com/ipfs/go-datastore/query"

	"github.com/AsynkronIT/protoactor-go/actor"
	blocks "github.com/ipfs/go-block-format"
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

const defaultTimeout = 30 * time.Second

type TreeStore interface {
	nodestore.NodeStore

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
	log.Debugf("new dag")

	storedTree := dag.NewDag(tip, ts)
	log.Debugf("new tree")

	tree, err := chaintree.NewChainTree(storedTree, nil, consensus.DefaultTransactors)
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

func (ts *IPLDTreeStore) GetNode(nodeCid cid.Cid) (*cbornode.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	blk, err := ts.blockApi.Get(ctx, nodeCid)
	if err == nil {
		return blockToCborNode(blk)
	}

	if err == format.ErrNotFound {
		return nil, nil
	}
	return nil, errors.Wrap(err, "error getting node")
}

func (ts *IPLDTreeStore) CreateNode(obj interface{}) (*cbornode.Node, error) {
	n, err := objToCbor(obj)
	if err != nil {
		return nil, errors.Wrap(err, "error converting to CBOR")
	}
	return n, ts.StoreNode(n)
}

func (ts *IPLDTreeStore) CreateNodeFromBytes(nodeBytes []byte) (*cbornode.Node, error) {
	sw := safewrap.SafeWrap{}
	n := sw.Decode(nodeBytes)
	if sw.Err != nil {
		return nil, errors.Wrap(sw.Err, "error decoding")
	}
	return n, ts.StoreNode(n)
}

func (ts *IPLDTreeStore) StoreNode(node *cbornode.Node) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	err := ts.blockApi.Add(ctx, node)
	if err != nil {
		return errors.Wrap(err, "error adding blocks")
	}

	actor.EmptyRootContext.Send(ts.publisher, node)

	return err
}

func (ts *IPLDTreeStore) DeleteNode(nodeCid cid.Cid) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return ts.blockApi.Remove(ctx, nodeCid)
}

func (ts *IPLDTreeStore) DeleteTree(tip cid.Cid) error {
	tipNode, err := ts.GetNode(tip)
	if err != nil {
		return fmt.Errorf("error getting tip: %v", err)
	}
	links := tipNode.Links()

	for _, link := range links {
		err := ts.DeleteTree(link.Cid)
		if err != nil {
			return fmt.Errorf("error deleting: %v", err)
		}
	}
	return ts.DeleteNode(tip)
}

func (ts *IPLDTreeStore) Resolve(tip cid.Cid, path []string) (val interface{}, remaining []string, err error) {
	node, err := ts.GetNode(tip)
	if err != nil {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("error getting node (%s)", tip.String()))
	}
	val, remaining, err = node.Resolve(path)
	if err != nil {
		switch err {
		case cbornode.ErrNoSuchLink:
			// If the link is just missing, then just return the whole path as remaining, with a nil value
			// instead of an error
			return nil, path, nil
		case cbornode.ErrNoLinks:
			// this means there was a simple value somewhere along the path
			// try resolving less of the path to find the existing boundary
			var err error
			for i := 1; i < len(path); i++ {
				val, _, err = node.Resolve(path[:len(path)-i])
				if err != nil {
					continue
				} else {
					// return the simple value and the rest of the path as remaining
					return val, path[len(path)-i:], nil
				}
			}
			if err != nil {
				return nil, nil, err
			}
		}
		return nil, nil, err
	}

	switch val := val.(type) {
	case *format.Link:
		linkNode, err := ts.GetNode(val.Cid)
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("error getting linked node (%s)", val.Cid.String()))
		}
		if linkNode != nil {
			return ts.Resolve(linkNode.Cid(), remaining)
		}
		return nil, remaining, nil
	default:
		return val, remaining, err
	}
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

		node, err := ts.GetNode(cid)
		if err != nil {
			return errors.Wrap(err, "error getting CID")
		}
		log.Infof("publishing %s", node.Cid().String())
		actor.EmptyRootContext.Send(ts.publisher, node)
	}
	return nil
}

func blockToCborNode(blk blocks.Block) (*cbornode.Node, error) {
	n, err := cbornode.DecodeBlock(blk)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding")
	}
	return n.(*cbornode.Node), nil
}

func objToCbor(obj interface{}) (node *cbornode.Node, err error) {
	sw := safewrap.SafeWrap{}
	node = sw.WrapObject(obj)
	if sw.Err != nil {
		return nil, fmt.Errorf("error wrapping: %v", sw.Err)
	}
	return
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
