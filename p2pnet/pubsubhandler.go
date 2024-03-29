package p2pnet

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

func newPubSub(cxt context.Context, host host.Host, topic string) *pubsub.PubSub {
	pubSubService, err := pubsub.NewGossipSub(cxt, host)
	if err != nil {
		fmt.Println("Error while establishing a pubsub service")
	} else {
		fmt.Println("Successfully established a new pubsubservice")
	}
	return pubSubService
}

func HandlePubSub(cxt context.Context, host host.Host, topic string) (*pubsub.Subscription, *pubsub.Topic) {
	pubSubServ := newPubSub(cxt, host, topic)
	pubSubTopic, err := pubSubServ.Join(topic)
	if err != nil {
		fmt.Println("Error while joining [", topic, "]")
	} else {
		fmt.Println("Successfull in joining [", topic, "]")
	}
	pubSubSubscription, err := pubSubServ.Subscribe(topic)
	if err != nil {
		fmt.Println("Error while subscribing to")
	}
	return pubSubSubscription, pubSubTopic
}
