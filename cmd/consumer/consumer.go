package main

import (
	"context"
	"fmt"
	"log"

	"github.com/apache/pulsar-client-go/pulsar"
)

func main() {

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: "pulsar://localhost:6650",
	})

	defer client.Close()

	chainName := "localterra"
	eventKind := "tm.event='NewBlock'"
	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            "persistent://terra/" + chainName + "/" + eventKind,
		SubscriptionName: "my-sub1",
		Type:             pulsar.Exclusive,
	})

	defer consumer.Close()

	msg, err := consumer.Receive(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Received message msgId: %#v -- content: '%s'\n",
		msg.ID(), string(msg.Payload()))
}
