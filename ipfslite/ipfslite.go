// Package ipfslite is a lightweight IPFS peer which runs the minimal setup to
// provide an `ipld.DAGService`, "Add" and "Get" UnixFS files from IPFS.
package ipfslite

import (
	"context"
	"fmt"
	"time"

	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	blockservice "github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	config "github.com/ipfs/go-ipfs-config"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/go-merkledag"
	dag "github.com/ipfs/go-merkledag"
	"github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	routing "github.com/libp2p/go-libp2p-routing"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/stats"
)

func init() {
	ipld.Register(cid.DagProtobuf, dag.DecodeProtobufBlock)
	ipld.Register(cid.Raw, dag.DecodeRawBlock)
	ipld.Register(cid.DagCBOR, cbor.DecodeBlock) // need to decode CBOR
}

var logger = logging.Logger("ipfslite")

type ipfsLiteStat struct {
	peers     int
	connected int
	id        peer.ID
	addrs     []multiaddr.Multiaddr
}

func (ils *ipfsLiteStat) Humanize() string {
	return fmt.Sprintf("%d known peers to (%s) / %d connected / addrs: %v", ils.peers, ils.id, ils.connected, ils.addrs)
}

// Config wraps configuration options for the Peer.
type Config struct {
	// The DAGService will not announce or retrieve blocks from the network
	Offline bool
}

// Peer is an IPFS-Lite peer. It provides a DAG service that can fetch and put
// blocks from/to the IPFS network.
type Peer struct {
	ctx context.Context

	cfg *Config

	ipld.DAGService
	bstore blockstore.Blockstore
	host   host.Host
	dht    *dht.IpfsDHT
}

// New creates an IPFS-Lite Peer. It uses the given datastore, libp2p Host and
// DHT. The Host and the DHT may be nil if config.Offline is set to true, as
// they are not used in that case. Peer implements the ipld.DAGService
// interface.
func New(
	ctx context.Context,
	store datastore.Batching,
	host host.Host,
	dht *dht.IpfsDHT,
	cfg *Config,
) (*Peer, error) {

	if cfg == nil {
		cfg = &Config{}
	}

	bs := blockstore.NewBlockstore(store)
	bs = blockstore.NewIdStore(bs)
	cachedbs, err := blockstore.CachedBlockstore(ctx, bs, blockstore.DefaultCacheOpts())
	if err != nil {
		return nil, err
	}

	var bserv blockservice.BlockService
	if cfg.Offline {
		bserv = blockservice.New(cachedbs, offline.Exchange(cachedbs))
	} else {
		bswapnet := network.NewFromIpfsHost(host, dht)
		bswap := bitswap.New(ctx, bswapnet, cachedbs)
		bserv = blockservice.New(cachedbs, bswap)
	}

	dags := merkledag.NewDAGService(bserv)
	return &Peer{
		ctx:        ctx,
		DAGService: dags,
		cfg:        cfg,
		bstore:     cachedbs,
		host:       host,
		dht:        dht,
	}, nil
}

func (p *Peer) Host() host.Host {
	return p.host
}

// Session returns a session-based NodeGetter.
func (p *Peer) Session(ctx context.Context) ipld.NodeGetter {
	ng := merkledag.NewSession(ctx, p.DAGService)
	if ng == p.DAGService {
		logger.Warning("DAGService does not support sessions")
	}
	return ng
}

// BlockStore offers access to the blockstore underlying the Peer's DAGService.
func (p *Peer) BlockStore() blockstore.Blockstore {
	return p.bstore
}

// HasBlock returns whether a given block is available locally. It is
// a shorthand for .Blockstore().Has().
func (p *Peer) HasBlock(c cid.Cid) (bool, error) {
	return p.BlockStore().Has(c)
}

func SetupLibp2p(
	ctx context.Context,
	hostKey crypto.PrivKey,
	secret []byte,
	listenAddrs []multiaddr.Multiaddr,
) (host.Host, *dht.IpfsDHT, error) {

	var idht *dht.IpfsDHT

	rHost, err := libp2p.New(
		ctx,
		libp2p.Identity(hostKey),
		libp2p.ListenAddrs(listenAddrs...),
		libp2p.NATPortMap(),
		libp2p.EnableAutoRelay(),
		// This weird construct is needed to enable AutoRelay
		// https://github.com/libp2p/go-libp2p/issues/487
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			// make the DHT with the given Host
			rting, err := dht.New(ctx, h)
			if err == nil {
				idht = rting
			}
			return rting, err
		}),
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error creating libp2p")
	}

	logger.Info("bootstraping")

	_, err = Bootstrap(rHost.(*routedhost.RoutedHost), idht, BootstrapConfigWithPeers(ipfslite.DefaultBootstrapPeers()))
	if err != nil {
		return nil, nil, errors.Wrap(err, "error bootstrapping")
	}

	go func() {
		tick := time.NewTicker(30 * time.Second)
		for {
			<-tick.C
			currentStats := &ipfsLiteStat{
				peers:     len(rHost.Peerstore().Peers()),
				connected: len(rHost.Network().Peers()),
				addrs:     rHost.Addrs(),
				id:        rHost.ID(),
			}
			stats.Stream.Publish(currentStats)
			logger.Infof("know about %d peers, connected to %d", currentStats.peers, currentStats.connected)
		}
	}()
	return rHost, idht, nil
}

// DefaultBootstrapPeers returns the default go-ipfs bootstrap peers (for use
// with NewLibp2pHost.
func DefaultBootstrapPeers() []peerstore.PeerInfo {
	// conversion copied from go-ipfs
	defaults, _ := config.DefaultBootstrapPeers()
	pinfos := make(map[peer.ID]*peerstore.PeerInfo)
	for _, bootstrap := range defaults {
		pinfo, ok := pinfos[bootstrap.ID()]
		if !ok {
			pinfo = new(peerstore.PeerInfo)
			pinfos[bootstrap.ID()] = pinfo
			pinfo.ID = bootstrap.ID()
		}

		pinfo.Addrs = append(pinfo.Addrs, bootstrap.Transport())
	}

	var peers []peerstore.PeerInfo
	for _, pinfo := range pinfos {
		peers = append(peers, *pinfo)
	}
	return peers
}
