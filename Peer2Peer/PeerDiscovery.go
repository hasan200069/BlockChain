package Peer2Peer

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type PeerManager struct {
	peers     map[string]bool
	peerMutex sync.RWMutex
	localPort int
}

func NewPeerManager(localPort int) *PeerManager {
	return &PeerManager{
		peers:     make(map[string]bool),
		localPort: localPort,
	}
}

func (pm *PeerManager) AddPeer(address string) {
	pm.peerMutex.Lock()
	defer pm.peerMutex.Unlock()
	pm.peers[address] = true
}

func (pm *PeerManager) RemovePeer(address string) {
	pm.peerMutex.Lock()
	defer pm.peerMutex.Unlock()
	delete(pm.peers, address)
}

func (pm *PeerManager) GetPeers() []string {
	pm.peerMutex.RLock()
	defer pm.peerMutex.RUnlock()
	peerList := make([]string, 0, len(pm.peers))
	for peer := range pm.peers {
		peerList = append(peerList, peer)
	}
	return peerList
}

func (pm *PeerManager) DiscoverPeers(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				pm.broadcastDiscovery()
				time.Sleep(5 * time.Minute)
			}
		}
	}()
}

func (pm *PeerManager) broadcastDiscovery() {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Println("Error getting network interfaces:", err)
		return
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagBroadcast == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					broadcastIP := getBroadcastIP(ipnet)
					pm.sendDiscoveryBroadcast(broadcastIP)
				}
			}
		}
	}
}

func getBroadcastIP(ipnet *net.IPNet) net.IP {
	ip := ipnet.IP.Mask(ipnet.Mask)
	broadcast := make(net.IP, len(ip))
	for i := range ip {
		broadcast[i] = ip[i] | ^ipnet.Mask[i]
	}
	return broadcast
}

func (pm *PeerManager) sendDiscoveryBroadcast(broadcastIP net.IP) {
	addr := fmt.Sprintf("%s:%d", broadcastIP.String(), pm.localPort)
	conn, err := net.Dial("udp", addr)
	if err != nil {
		log.Println("Discovery broadcast error:", err)
		return
	}
	defer conn.Close()

	message := []byte("BLOCKCHAIN_PEER_DISCOVERY")
	_, err = conn.Write(message)
	if err != nil {
		log.Println("Error sending discovery message:", err)
	}
}

func (pm *PeerManager) IsPeerRegistered(address string) bool {
    pm.peerMutex.RLock()
    defer pm.peerMutex.RUnlock()
    _, exists := pm.peers[address]
    return exists
}
