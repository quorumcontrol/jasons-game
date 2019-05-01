package network

import (
	"context"
	"fmt"
	"time"

	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	cbornode "github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
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

type IPLDTreeStore struct {
	TreeStore

	blockApi    format.DAGService
	keyValueApi datastore.Batching
}

func NewIPLDTreeStore(blockApi format.DAGService, keyValueApi datastore.Batching) *IPLDTreeStore {
	return &IPLDTreeStore{
		blockApi:    blockApi,
		keyValueApi: keyValueApi,
	}
}

func (ts *IPLDTreeStore) GetTree(did string) (*consensus.SignedChainTree, error) {
	tip, err := ts.getTip(did)
	if err != nil {
		return nil, errors.Wrap(err, "error getting tip")
	}
	storedTree := dag.NewDag(tip, ts)

	tree, err := chaintree.NewChainTree(storedTree, nil, consensus.DefaultTransactors)
	if err != nil {
		return nil, errors.Wrap(err, "error creating chaintree")
	}

	sigs, err := ts.getSignatures(did)
	if err != nil {
		return nil, errors.Wrap(err, "error getting signatures")
	}

	return &consensus.SignedChainTree{
		ChainTree:  tree,
		Signatures: sigs,
	}, nil
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return n, ts.blockApi.Add(ctx, n)
}

func (ts *IPLDTreeStore) CreateNodeFromBytes(nodeBytes []byte) (*cbornode.Node, error) {
	sw := safewrap.SafeWrap{}
	n := sw.Decode(nodeBytes)
	if sw.Err != nil {
		return nil, errors.Wrap(sw.Err, "error decoding")
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return n, ts.blockApi.Add(ctx, n)
}

func (ts *IPLDTreeStore) StoreNode(node *cbornode.Node) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return ts.blockApi.Add(ctx, node)
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
			return nil, nil, errors.Wrap(err, fmt.Sprintf("error getting linked node (%s)", linkNode.Cid().String()))
		}
		if linkNode != nil {
			return ts.Resolve(linkNode.Cid(), remaining)
		}
		return nil, remaining, nil
	default:
		return val, remaining, err
	}
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

func (ts *IPLDTreeStore) getTip(did string) (cid.Cid, error) {
	tip, err := ts.keyValueApi.Get(didStoreKey(did))
	if err != nil {
		return cid.Undef, errors.Wrap(err, "error getting key")
	}
	tipCid, err := cid.Cast(tip)
	if err != nil {
		return cid.Undef, errors.Wrap(err, "error casting tip")
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
