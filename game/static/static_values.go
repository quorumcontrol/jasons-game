package static

import (
	"context"

	"github.com/quorumcontrol/jasons-game/network"
)

const STATIC_DID = "did:tupelo:0xB5F67728cdb4E809aBDc7386245e58b782453863"

func Get(net network.Network, key string) (string, error) {
	tree, err := net.GetTree(STATIC_DID)
	if err != nil || tree == nil {
		return "", err
	}

	val, _, err := tree.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "values", key})
	if err != nil || val == nil {
		return "", err
	}

	return val.(string), nil
}

func GetAll(net network.Network) (map[string]string, error) {
	ret := make(map[string]string)
	tree, err := net.GetTree(STATIC_DID)
	if err != nil || tree == nil {
		return ret, err
	}

	val, _, err := tree.ChainTree.Dag.Resolve(context.Background(), []string{"tree", "data", "jasons-game", "values"})
	if err != nil || val == nil {
		return ret, err
	}

	for k, v := range val.(map[string]interface{}) {
		ret[k] = v.(string)
	}
	return ret, nil
}
