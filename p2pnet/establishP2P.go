package p2pnet

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	tls "github.com/libp2p/go-libp2p-tls"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

func EstablishP2P() (context.Context, host.Host) {
	prvkey := GenerateKey()
	identity := libp2p.Identity(prvkey)
	nat := libp2p.NATPortMap()
	holepunch := libp2p.EnableHolePunching()
	tlstramsport, err := tls.New(prvkey)
	if err != nil {
		fmt.Println("Error while setting up a tls transport")
	}
	security := libp2p.Security(tls.ID, tlstramsport)
	transport := libp2p.Transport(tcp.NewTCPTransport)
	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"), identity, nat, security, transport, holepunch)
	if err != nil {
		fmt.Println("Error while creating a new node")
	} else {
		fmt.Println("Successfully created a new node")
	}
	ctx := context.Background()
	return ctx, host
}
