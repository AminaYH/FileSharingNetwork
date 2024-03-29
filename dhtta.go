package main

import (
	plog "FileSharingNetwork/fileshare"
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"log"
	"sync"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"

	"strings"
)

func SourceNode() host.Host {
	node, err := libp2p.New()
	if err != nil {
		panic(err)
	}

	return node
}
func NewDh(ctx context.Context, host host.Host, Peers []multiaddr.Multiaddr) (*dht.IpfsDHT, error) {
	var options []dht.Option

	if len(Peers) == 0 {
		options = append(options, dht.Mode(dht.ModeServer))
	}

	thisdht, err := dht.New(ctx, host, options...)
	if err != nil {
		return nil, err
	}
	if err = thisdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	for _, peerAddr := range Peers {
		peerinformations, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			return nil, err
		}
		wg.Add(1)

		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinformations); err != nil {
				log.Printf("Error while connecting to node %q: %-v", peerinformations, err)
			} else {
				log.Printf("Connection established with bootstrap node: %q", *peerinformations)
			}
		}()
	}
	wg.Wait()

	return thisdht, nil
}
func DestinationNode() host.Host {

	listenAddr := "/ip4/172.17.0.1/tcp/9090"
	node, err := libp2p.New(libp2p.ListenAddrStrings(listenAddr))
	if err != nil {
		panic(err)
	}

	return node
}
func connectToNodeFromSource(sourceNode host.Host, targetNode host.Host) {
	targetNodeAddressInfo := host.InfoFromHost(targetNode)
	err := sourceNode.Connect(context.Background(), *targetNodeAddressInfo)
	if err != nil {
		panic(err)
	}
}
func countSourceNodePeers(sourceNode host.Host) int {
	return len(sourceNode.Network().Peers())
}
func printNodeID(host host.Host) {
	println(fmt.Sprintf("ID: %s", host.ID().String()))
}

func printNodeAddresses(host host.Host) {
	addressesString := make([]string, 0)
	for _, address := range host.Addrs() {
		addressesString = append(addressesString, address.String())
	}

	println(fmt.Sprintf("Multiaddresses: %s", strings.Join(addressesString, ", ")))
}
func createNodeWithMultiaddr(ctx context.Context, listenAddress multiaddr.Multiaddr) (host.Host, error) {
	// Create a new libp2p node specifying the listen address
	node, err := libp2p.New(libp2p.ListenAddrStrings(listenAddress.String()))
	if err != nil {
		return nil, err
	}
	return node, nil
}

func main() {
	ctx := context.Background()

	sourceNode := SourceNode()
	println("-- SOURCE NODE INFORMATION --")
	printNodeID(sourceNode)
	printNodeAddresses(sourceNode)

	targetNode := DestinationNode()
	println("-- TARGET NODE INFORMATION --")
	printNodeID(targetNode)
	printNodeAddresses(targetNode)

	connectToNodeFromSource(sourceNode, targetNode)
	fmt.Printf("##########################\n")

	// view host details and addresses
	fmt.Printf("host ID %s\n", sourceNode.ID())
	fmt.Printf("following are the assigned addresses\n")
	for _, addr := range sourceNode.Addrs() {
		fmt.Printf("%s\n", addr.String())
	}
	fmt.Printf("\n")

	// create a new PubSub service using the GossipSub router

	_, err := pubsub.NewGossipSub(ctx, sourceNode)
	if err != nil {
		panic(err)
	}
	var bootstrapPeers []multiaddr.Multiaddr

	Adress := []multiaddr.Multiaddr{
		multiaddrString("/ip4/172.17.0.1/tcp/4001"),
		multiaddrString("/ip4/172.17.0.1/tcp/4000"),
		multiaddrString("/ip4/172.17.0.1/tcp/5000"),
		multiaddrString("/ip4/172.17.0.1/tcp/4500"),
		multiaddrString("/ip4/172.17.0.1/tcp/4600"),
	}
	for _, addr := range Adress {
		node, err := createNodeWithMultiaddr(ctx, addr)
		if err != nil {
			panic(err)
		}
		fmt.Printf("*********\n")
		peerAddr := addr.Encapsulate(multiaddrString(fmt.Sprintf("/ipfs/%s", node.ID())))

		// Append the bootstrap peer address to the list
		bootstrapPeers = append(bootstrapPeers, peerAddr)
		// Print the ID and addresses of the created node
		fmt.Println("Node ID:", node.ID())
		fmt.Println("Node Addresses:")
		for _, addr := range node.Addrs() {
			fmt.Println(addr)
		}
		fmt.Println("---------------------------------------")
	}

	dht, err := NewDh(ctx, sourceNode, bootstrapPeers)
	if err != nil {
		panic(err)
	}

	println(dht)

	fmt.Printf("##########################\n")

	println(fmt.Sprintf("Source node peers: %d", countSourceNodePeers(sourceNode)))
}
func multiaddrString(addr string) multiaddr.Multiaddr {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		panic(err)
	}
	return maddr
}

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
