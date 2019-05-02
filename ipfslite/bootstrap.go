package ipfslite

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	goprocess "github.com/jbenet/goprocess"
	procctx "github.com/jbenet/goprocess/context"
	periodicproc "github.com/jbenet/goprocess/periodic"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	lgbl "github.com/libp2p/go-libp2p-loggables"
	inet "github.com/libp2p/go-libp2p-net"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"
)

func convertPeers(peers []string) []pstore.PeerInfo {
	pinfos := make([]pstore.PeerInfo, len(peers))
	for i, peer := range peers {
		maddr := ma.StringCast(peer)
		p, err := pstore.InfoFromP2pAddr(maddr)
		if err != nil {
			logger.Fatal(err)
		}
		pinfos[i] = *p
	}
	return pinfos
}

// ErrNotEnoughBootstrapPeers signals that we do not have enough bootstrap
// peers to bootstrap correctly.
var ErrNotEnoughBootstrapPeers = errors.New("not enough bootstrap peers to bootstrap")

// BootstrapConfig specifies parameters used in an IpfsNode's network
// bootstrapping process.
type BootstrapConfig struct {

	// MinPeerThreshold governs whether to bootstrap more connections. If the
	// node has less open connections than this number, it will open connections
	// to the bootstrap nodes. From there, the routing system should be able
	// to use the connections to the bootstrap nodes to connect to even more
	// peers. Routing systems like the IpfsDHT do so in their own Bootstrap
	// process, which issues random queries to find more peers.
	MinPeerThreshold int

	// Period governs the periodic interval at which the node will
	// attempt to bootstrap. The bootstrap process is not very expensive, so
	// this threshold can afford to be small (<=30s).
	Period time.Duration

	// ConnectionTimeout determines how long to wait for a bootstrap
	// connection attempt before cancelling it.
	ConnectionTimeout time.Duration

	// BootstrapPeers is a function that returns a set of bootstrap peers
	// for the bootstrap process to use. This makes it possible for clients
	// to control the peers the process uses at any moment.
	BootstrapPeers func() []pstore.PeerInfo
}

// DefaultBootstrapConfig specifies default sane parameters for bootstrapping.
var DefaultBootstrapConfig = BootstrapConfig{
	MinPeerThreshold:  4,
	Period:            2 * time.Second,
	ConnectionTimeout: (30 * time.Second) / 3,
}

func BootstrapConfigWithPeers(pis []pstore.PeerInfo) BootstrapConfig {
	cfg := DefaultBootstrapConfig
	cfg.BootstrapPeers = func() []pstore.PeerInfo {
		return pis
	}
	return cfg
}

// Bootstrap kicks off IpfsNode bootstrapping. This function will periodically
// check the number of open connections and -- if there are too few -- initiate
// connections to well-known bootstrap peers. It also kicks off subsystem
// bootstrapping (i.e. routing).
func Bootstrap(h *rhost.RoutedHost, routing *dht.IpfsDHT, cfg BootstrapConfig) (io.Closer, error) {

	// make a signal to wait for one bootstrap round to complete.
	doneWithRound := make(chan struct{})

	// the periodic bootstrap function -- the connection supervisor
	periodic := func(worker goprocess.Process) {
		ctx := procctx.OnClosingContext(worker)
		defer logger.EventBegin(ctx, "periodicBootstrap", h.ID()).Done()

		if err := bootstrapRound(ctx, h, cfg); err != nil {
			logger.Event(ctx, "bootstrapError", h.ID(), lgbl.Error(err))
			logger.Debugf("%s bootstrap error: %s", h.ID(), err)
		}

		<-doneWithRound
	}

	// kick off the node's periodic bootstrapping
	proc := periodicproc.Tick(cfg.Period, periodic)
	proc.Go(periodic) // run one right now.

	// kick off Routing.Bootstrap
	if routing != nil {
		ctx := procctx.OnClosingContext(proc)
		if err := routing.Bootstrap(ctx); err != nil {
			proc.Close()
			return nil, err
		}
	}

	doneWithRound <- struct{}{}
	close(doneWithRound) // it no longer blocks periodic
	return proc, nil
}

func bootstrapRound(ctx context.Context, h *rhost.RoutedHost, cfg BootstrapConfig) error {
	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout)
	defer cancel()
	id := h.ID()

	// get bootstrap peers from config. retrieving them here makes
	// sure we remain observant of changes to client configuration.
	peers := cfg.BootstrapPeers()

	// determine how many bootstrap connections to open
	connected := h.Network().Peers()
	if len(connected) >= cfg.MinPeerThreshold {
		logger.Event(ctx, "bootstrapSkip", id)
		return nil
	}
	numToDial := cfg.MinPeerThreshold - len(connected)

	// filter out bootstrap nodes we are already connected to
	var notConnected []pstore.PeerInfo
	for _, p := range peers {
		if h.Network().Connectedness(p.ID) != inet.Connected {
			notConnected = append(notConnected, p)
		}
	}

	// if connected to all bootstrap peer candidates, exit
	if len(notConnected) < 1 {
		logger.Debugf("%s no more bootstrap peers to create %d connections", id, numToDial)
		return ErrNotEnoughBootstrapPeers
	}

	// connect to a random susbset of bootstrap candidates
	randSubset := randomSubsetOfPeers(notConnected, numToDial)

	defer logger.EventBegin(ctx, "bootstrapStart", id).Done()
	logger.Debugf("%s bootstrapping to %d nodes: %s", id, numToDial, randSubset)
	return bootstrapConnect(ctx, h, randSubset)
}

func bootstrapConnect(ctx context.Context, ph host.Host, peers []pstore.PeerInfo) error {
	if len(peers) < 1 {
		return ErrNotEnoughBootstrapPeers
	}

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {

		// performed asynchronously because when performed synchronously, if
		// one `Connect` call hangs, subsequent calls are more likely to
		// fail/abort due to an expiring context.
		// Also, performed asynchronously for dial speed.

		wg.Add(1)
		go func(p pstore.PeerInfo) {
			defer wg.Done()
			defer logger.EventBegin(ctx, "bootstrapDial", ph.ID(), p.ID).Done()
			logger.Debugf("%s bootstrapping to %s", ph.ID(), p.ID.Pretty())

			ph.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				logger.Event(ctx, "bootstrapDialFailed", p.ID)
				logger.Debugf("failed to bootstrap with %v: %s", p.ID, err)
				errs <- err
				return
			}
			logger.Event(ctx, "bootstrapDialSuccess", p.ID)
			logger.Infof("bootstrapped with %v", p.ID)
		}(p)
	}
	wg.Wait()

	// our failure condition is when no connection attempt succeeded.
	// So drain the errs channel, counting the results.
	close(errs)
	count := 0
	var err error
	for err = range errs {
		if err != nil {
			count++
		}
	}
	if count == len(peers) {
		return fmt.Errorf("failed to bootstrap. %s", err)
	}
	return nil
}

func randomSubsetOfPeers(in []pstore.PeerInfo, max int) []pstore.PeerInfo {
	n := len(in)
	if n > max {
		n = max
	}
	var out []pstore.PeerInfo
	for _, val := range rand.Perm(len(in)) {
		out = append(out, in[val])
		if len(out) >= n {
			break
		}
	}
	return out
}
