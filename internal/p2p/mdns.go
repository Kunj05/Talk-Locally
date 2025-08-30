package p2p

import (
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

// Called when a new peer is found via mDNS
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.PeerChan <- pi
}

// InitMDNS sets up mDNS peer discovery and returns a channel that emits new peers
func InitMDNS(peerhost host.Host, rendezvous string) chan peer.AddrInfo {
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)

	// Create the mDNS service
	ser := mdns.NewMdnsService(peerhost, rendezvous, n)
	if err := ser.Start(); err != nil {
		panic(err) //consider handling more gracefully in production
	}
	return n.PeerChan
}
