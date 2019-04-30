package network

import (
	"context"
	"fmt"
	"time"

	discovery "github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	inet "github.com/libp2p/go-libp2p-net"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

const nameSpace = "jasons-game"

const maxConnected = 300

type jasonsDiscoverer struct {
	host       host.Host
	discoverer *discovery.RoutingDiscovery
}

func newJasonsDiscoverer(h host.Host, dht *dht.IpfsDHT) *jasonsDiscoverer {
	return &jasonsDiscoverer{
		host:       h,
		discoverer: discovery.NewRoutingDiscovery(dht),
	}
}

func (td *jasonsDiscoverer) doDiscovery(ctx context.Context) error {
	if err := td.constantlyAdvertise(ctx); err != nil {
		return fmt.Errorf("error advertising: %v", err)
	}
	if err := td.findPeers(ctx); err != nil {
		return fmt.Errorf("error finding peers: %v", err)
	}
	return nil
}

func (td *jasonsDiscoverer) findPeers(ctx context.Context) error {
	peerChan, err := td.discoverer.FindPeers(ctx, nameSpace)
	if err != nil {
		return fmt.Errorf("error findPeers: %v", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case peerInfo := <-peerChan:
				td.handleNewPeerInfo(ctx, peerInfo)
			}
		}
	}()
	return nil
}

func (td *jasonsDiscoverer) handleNewPeerInfo(ctx context.Context, p pstore.PeerInfo) {
	host := td.host
	if host.Network().Connectedness(p.ID) == inet.Connected {
		return // we are already connected
	}

	connected := host.Network().Peers()
	if len(connected) > maxConnected {
		return // we already are connected to more than we need
	}

	if p.ID == "" {
		return // empty id
	}

	// log.Debugf("new peer: %s", p.ID)

	// do the connection async because connect can hang
	go func() {
		// not actually positive that TTL is correct, but it seemed the most appropriate
		host.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.ProviderAddrTTL)
		if err := host.Connect(ctx, p); err != nil {
			// log.Errorf("error connecting to  %s %v: %v", p.ID, p, err)
		}
	}()
}

func (td *jasonsDiscoverer) constantlyAdvertise(ctx context.Context) error {
	dur, err := td.discoverer.Advertise(ctx, nameSpace)
	if err != nil {
		return err
	}
	go func() {
		after := time.After(dur)
		select {
		case <-ctx.Done():
			return
		case <-after:
			if err := td.constantlyAdvertise(ctx); err != nil {
				// log.Errorf("error constantly advertising: %v", err)
			}
		}
	}()
	return nil
}
