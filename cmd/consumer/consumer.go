package main

import (
	"context"
	"fmt"
	"log"
	"rpc_watcher/rpcwatcher/avro"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/terra-money/mantlemint/block_feed"
)

func main() {

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: "pulsar://localhost:6650",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	b := &block_feed.BlockResult{}
	schema := avro.GenerateAvroSchema(&block_feed.BlockResult{})

	consumerJS := pulsar.NewJSONSchema(schema, nil)

	chainName := "localterra"
	eventKind := "tm.event='NewBlock'"
	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:                       "persistent://terra/" + chainName + "/" + eventKind,
		SubscriptionName:            "my-sub1",
		Type:                        pulsar.Exclusive,
		Schema:                      consumerJS,
		SubscriptionInitialPosition: pulsar.SubscriptionPositionLatest,
	})
	if err != nil {
		log.Fatal(err)
	}

	defer consumer.Close()

	msg, err := consumer.Receive(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	err = msg.GetSchemaValue(&b)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Received message msgId: %#v -- content:\n '%+v'\n",
		msg.ID(), b)
}
