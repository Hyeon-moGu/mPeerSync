package peer

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	peerdiscovery "github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"mPeerSync/util"
)

type DiscoveryNotifee struct {
	Host               host.Host
	OnPeerConnected    func(peer.ID)
	IsAlreadyConnected func(peer.ID) bool
	MarkConnected      func(peer.ID)
	UnmarkConnected    func(peer.ID)
}

const discoveryServiceTag = "mPeerSync"

func (n *DiscoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	util.Info("Discovered peer: %s", pi.ID)

	if n.IsAlreadyConnected(pi.ID) {
		util.Warn("Peer %s is already connected, skipping", pi.ID)
		return
	}

	n.MarkConnected(pi.ID)

	go func() {
		retryInterval := 3 * time.Second
		maxRetries := 3

		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := n.Host.Connect(context.Background(), pi)
			if err == nil {
				util.Success("Connected to peer %s", pi.ID)
				n.OnPeerConnected(pi.ID)
				return
			}
			util.Fail("[RETRY %d/%d] Failed to connect to %s: %v", attempt, maxRetries, pi.ID, err)
			time.Sleep(retryInterval)
		}

		n.UnmarkConnected(pi.ID)
		util.Fail("GIVE UP: Could not connect to peer %s after %d attempts", pi.ID, maxRetries)
	}()
}

func SetupMDNS(ctx context.Context, h host.Host, notifee *DiscoveryNotifee) error {
	svc := peerdiscovery.NewMdnsService(h, discoveryServiceTag, notifee)
	return svc.Start()
}
