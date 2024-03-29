package p2pnet

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

func GenerateKey() crypto.PrivKey {
	privkey, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		fmt.Println("Error while generating the key pair")
	} else {
		fmt.Println("Sucessfull in generating the pair")
		keyval, _ := peer.IDFromPrivateKey(privkey)
		fmt.Println("PUBLIC KEY:", keyval)
	}
	return privkey
}

// func GetID() peer.ID {
// 	pk := generateID()
// 	peerId, err := peer.IDFromPrivateKey(pk)
// 	if err != nil {
// 		fmt.Println("Error while creating peerid from public key")
// 	}
// 	return peerId
// }
