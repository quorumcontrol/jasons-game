package cache

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	packr "github.com/gobuffalo/packr/v2"
	format "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("jgcache")

func Load(store format.NodeAdder) error {
	box := packr.New("datacache", "./out")
	gzippedNodes, err := box.Find("nodes.gz")
	if err != nil {
		err = errors.Wrap(err, "could not find nodes.gz")
		log.Error(err)
		return err
	}

	gz, err := gzip.NewReader(bytes.NewReader(gzippedNodes))
	if err != nil {
		err = errors.Wrap(err, "gzip reader error")
		log.Error(err)
		return err
	}

	startTime := time.Now()
	sw := &safewrap.SafeWrap{}
	loadedCount := 0
	i := 0

	for {
		i++

		dataSize := make([]byte, 8)
		_, err := io.ReadFull(gz, dataSize)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(errors.Wrap(err, fmt.Sprintf("error importing node prefix %d", i)))
			continue
		}

		dataSizeInt := binary.LittleEndian.Uint64(dataSize)
		data := make([]byte, dataSizeInt)
		_, err = io.ReadFull(gz, data)
		if err != nil {
			log.Error(errors.Wrap(err, fmt.Sprintf("error importing node %d", i)))
			continue
		}

		node := sw.Decode(data)
		if sw.Err != nil {
			log.Error(errors.Wrap(sw.Err, fmt.Sprintf("error decoding node %d", i)))
			continue
		}

		err = store.Add(context.Background(), node)
		if err != nil {
			log.Error(errors.Wrap(sw.Err, fmt.Sprintf("error adding node %d", i)))
			continue
		}

		loadedCount++
	}

	log.Infof("Loaded %d nodes from cache in %s", loadedCount, time.Since(startTime))
	return nil
}

func Export(net network.Network) error {
	ctx := context.Background()

	didsFile, err := ioutil.ReadFile("dids.txt")
	if err != nil {
		return errors.Wrap(err, "error reading dids.txt")
	}
	dids := strings.Split(string(didsFile), "\n")
	// trim last DID if it's blank (from trailing newline in source file)
	if len(dids) > 0 && dids[len(dids)-1] == "" {
		dids = dids[:len(dids)-1]
	}

	log.Infof("exporting %d dids", len(dids))

	filename := "./cache/out/nodes.gz"
	f, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "error opening "+filename)
	}
	gz := gzip.NewWriter(f)

	defer func() {
		if err := gz.Close(); err != nil {
			log.Error(errors.Wrap(err, "error closing gzip"))
		}
		if err := f.Close(); err != nil {
			log.Error(errors.Wrap(err, "error closing file"))
		}
	}()

	for _, did := range dids {
		log.Infof("exporting %s", did)

		tree, err := net.GetTree(did)
		if err != nil {
			return errors.Wrap(err, "error loading "+did)
		}

		nodes, err := tree.ChainTree.Dag.Nodes(ctx)
		if err != nil {
			return errors.Wrap(err, "error loading nodes for "+did)
		}

		totalBytes := 0

		for _, node := range nodes {
			byteLen := make([]byte, 8)
			data := node.RawData()
			totalBytes = totalBytes + len(data)
			binary.LittleEndian.PutUint64(byteLen, uint64(len(data)))
			_, err := gz.Write(byteLen)
			if err != nil {
				return errors.Wrap(err, "error writing bytes prefix for "+did)
			}
			_, err = gz.Write(data)
			if err != nil {
				return errors.Wrap(err, "error writing bytes for "+did)
			}
		}

		log.Infof("exported %s - %d nodes - %d bytes", did, len(nodes), totalBytes)
	}

	log.Infof("exported %d dids", len(dids))
	return nil
}
