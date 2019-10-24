package trees

import (
	"context"
	"fmt"

	format "github.com/ipfs/go-ipld-format"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
)

func NodesAsBytes(ctx context.Context, tree *chaintree.ChainTree) ([][]byte, error) {
	nodes, err := tree.Dag.Nodes(ctx)
	if err != nil {
		return nil, err
	}

	asBytes := make([][]byte, len(nodes))

	for i, node := range nodes {
		asBytes[i] = node.RawData()
	}

	return asBytes, nil
}

func NodesFromBytes(bytes [][]byte) ([]format.Node, error) {
	nodes := make([]format.Node, len(bytes))

	sw := &safewrap.SafeWrap{}

	for i, nodeBytes := range bytes {
		nodes[i] = sw.Decode(nodeBytes)

		if sw.Err != nil {
			return nodes, fmt.Errorf("Error decoding node bytes: %v", sw.Err)
		}
	}

	return nodes, nil
}

func LoadNodesFromBytes(ctx context.Context, store nodestore.DagStore, bytes [][]byte) error {
	if bytes == nil || len(bytes) == 0 {
		return nil
	}

	nodes, err := NodesFromBytes(bytes)
	if err != nil {
		return err
	}

	return store.AddMany(ctx, nodes)
}
