package config

import (
	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	badger "github.com/ipfs/go-ds-badger"
	"github.com/pkg/errors"
)

func LocalDataStore(path string) (datastore.Batching, error) {
	ds, err := badger.NewDatastore(path, &badger.DefaultOptions)
	if err != nil {
		return nil, errors.Wrap(err, "error creating store")
	}

	return ds, nil
}

func MemoryDataStore() datastore.Batching {
	return dssync.MutexWrap(datastore.NewMapDatastore())
}
