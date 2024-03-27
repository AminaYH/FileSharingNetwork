package main

import (
	"context"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
	"strings"
)

func createNode(ctx context.Context) (host.Host, *dht.IpfsDHT, error) {
	// Create a new libp2p Host
	node, err := libp2p.New(
	// include other options as needed
	)
	if err != nil {
		return nil, nil, err
	}

	// Create a new DHT instance
	kademliaDHT, err := dht.New(ctx, node)
	if err != nil {
		return nil, nil, err
	}

	return node, kademliaDHT, nil
}

func bootstrapDHT(ctx context.Context, node host.Host, kademliaDHT *dht.IpfsDHT) error {
	// Bootstrap the DHT
	err := kademliaDHT.Bootstrap(ctx)
	if err != nil {
		return err
	}

	// Connect to bootstrap nodes (example addresses)
	bootstrapPeers := dht.DefaultBootstrapPeers

	for _, peerAddr := range bootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		if err := node.Connect(ctx, *peerinfo); err != nil {
			fmt.Println("Bootstrap connection failed:", err)
		}
	}

	return nil
}

func stringToCid(contentID string) (cid.Cid, error) {
	// Convert the contentID into a hash using the identity function.
	// For real applications, consider a different hash function.
	mh, err := multihash.Sum([]byte(contentID), multihash.IDENTITY, -1)
	if err != nil {
		return cid.Cid{}, err
	}

	// Construct a CID using the multihash.
	c := cid.NewCidV1(cid.Raw, mh)
	return c, nil
}

func findPeers(ctx context.Context, kademliaDHT *dht.IpfsDHT, contentID string) {
	// Convert the contentID into a CID.
	c, err := stringToCid(contentID)
	if err != nil {
		fmt.Println("Error converting contentID to CID:", err)
		return
	}

	// Use the CID to find providers.
	peers, err := kademliaDHT.FindProviders(ctx, c)
	if err != nil {
		fmt.Println("Finding peers error:", err)
		return
	}

	for _, p := range peers {
		fmt.Println("Found peer:", p.ID)
	}
}

func createTargetNode() host.Host {
	node, err := libp2p.New(
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/8007",
		),
	)
	if err != nil {
		panic(err)
	}
	return node
}

func connectToTargetNode(sourceNode host.Host, targetNode host.Host) {
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
func createHost(ctx context.Context) (host.Host, error) {
	return libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"),
	)
}

func main() {
	ctx := context.Background()

	// Use createNode to create a new libp2p node and a new DHT instance.
	node, kadDHT, err := createNode(ctx)
	if err != nil {
		panic(err)
	}
	defer node.Close()

	// Example of bootstrapping the DHT (you can implement bootstrapDHT as needed).
	err = bootstrapDHT(ctx, node, kadDHT)
	if err != nil {
		fmt.Println("Bootstrap error:", err)
		return
	}

	// Use the newly created node for operations.
	// For example, printing the node's information.
	printNodeID(node)
	printNodeAddresses(node)

	// Example usage of findPeers function.
	contentID := "exampleContentID"
	findPeers(ctx, kadDHT, contentID)
}
