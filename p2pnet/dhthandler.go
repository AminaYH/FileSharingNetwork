package p2pnet

import (
	"context"
	"fmt"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
)

func initDHT(ctx context.Context, host host.Host) *dht.IpfsDHT {
	kad_dht, err := dht.New(ctx, host)
	if err != nil {
		fmt.Println("Error while creating the new DHT")
	} else {
		fmt.Println("Successfully created the new DHT")
	}
	return kad_dht
}

func bootstrapDHT(ctx context.Context, host host.Host, kad_dht *dht.IpfsDHT) {
	err := kad_dht.Bootstrap(ctx)
	if err != nil {
		fmt.Println("Error while setting the DHT in bootstrap mode")
	} else {
		fmt.Println("Successfully set the DHT in bootstrap mode")
	}
	for _, bootstrapPeers := range dht.GetDefaultBootstrapPeerAddrInfos() {
		err := host.Connect(ctx, bootstrapPeers)
		if err != nil {
			fmt.Println("Error while connecting to :", bootstrapPeers.ID)
		} else {
			fmt.Println("Successfully connected to :", bootstrapPeers.ID)
		}
	}
	fmt.Println("Done with all connections to DHT Bootstrap peers")
}

func HandleDHT(ctx context.Context, host host.Host) *dht.IpfsDHT {
	kad_dht := initDHT(ctx, host)
	bootstrapDHT(ctx, host, kad_dht)
	return kad_dht
}
