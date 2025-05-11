package peer

import (
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

type PeerManager struct {
	connectedPeers   []peer.ID
	connectedPeerMap sync.Map
	mutex            sync.Mutex
}

func NewPeerManager() *PeerManager {
	return &PeerManager{}
}

func (pm *PeerManager) IsConnected(pid peer.ID) bool {
	_, ok := pm.connectedPeerMap.Load(pid)
	return ok
}

func (pm *PeerManager) MarkConnected(pid peer.ID) {
	pm.connectedPeerMap.Store(pid, struct{}{})
}

func (pm *PeerManager) UnmarkConnected(pid peer.ID) {
	pm.connectedPeerMap.Delete(pid)
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	for i, id := range pm.connectedPeers {
		if id == pid {
			pm.connectedPeers = append(pm.connectedPeers[:i], pm.connectedPeers[i+1:]...)
			break
		}
	}
}

func (pm *PeerManager) AddPeer(pid peer.ID) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.connectedPeers = append(pm.connectedPeers, pid)
}

func (pm *PeerManager) ListPeers() []peer.ID {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	return append([]peer.ID(nil), pm.connectedPeers...)
}
