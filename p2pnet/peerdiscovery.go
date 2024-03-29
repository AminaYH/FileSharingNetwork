package p2pnet

import (
	"context"
	"fmt"

	plog "pubsubfilesharing/fileshare"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

func DiscoverPeers(ctx context.Context, host host.Host, service string, kad_dht *dht.IpfsDHT) {
	peerlog := plog.OpenPeerConnectionLog()
	constat := plog.OpenConnectionStatusLog()
	routingDiscovery := drouting.NewRoutingDiscovery(kad_dht)
	dutil.Advertise(ctx, routingDiscovery, service)
	fmt.Println("Successfull in advertising service")
	connectedPeers := []peer.AddrInfo{}
	isAlreadyConnected := false
	for len(connectedPeers) < 20 {
		fmt.Fprintln(constat, "Currently connected to", len(connectedPeers), "out of 5 [for service", service, "]")
		fmt.Fprintln(constat, "TOTAL CONNECTIONS : ", len(host.Network().Conns()))
		peerChannel, err := routingDiscovery.FindPeers(ctx, service)
		if err != nil {
			fmt.Println("Error while finding some peers for service :", service)
		} else {
			fmt.Fprintln(constat, "Successfull in finding some peers")
		}
		for peerAddr := range peerChannel {

			if peerAddr.ID == host.ID() {
				continue
			}
			for _, connPeers := range connectedPeers {
				if connPeers.ID == peerAddr.ID {
					fmt.Fprintln(peerlog, "Already have a connection with ", peerAddr.ID)
					isAlreadyConnected = true
					break
				}
			}
			if isAlreadyConnected {
				isAlreadyConnected = false
				continue
			}

			err := host.Connect(ctx, peerAddr)
			if err != nil {
				fmt.Fprintln(peerlog, "Error while connecting to peer ", peerAddr.ID)
			} else {
				fmt.Println("Successfull in connecting to peer :", peerAddr.ID.Pretty()[len(peerAddr.ID.Pretty())-6:len(peerAddr.ID.Pretty())])
				connectedPeers = append(connectedPeers, peerAddr)
				fmt.Println("Currently connected to", len(connectedPeers), "out of 5 [for service", service, "]")
			}
		}
	}
}
