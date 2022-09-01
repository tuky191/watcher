package main

import (
	"context"
	"fmt"
	"log"
	"rpc_watcher/rpcwatcher/avro"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/davecgh/go-spew/spew"
	"github.com/terra-money/mantlemint/block_feed"
)

func main() {

	schema := avro.GenerateAvroSchema(&block_feed.BlockResult{})
	//consumerJS := pulsar.NewJSONSchema(schema, nil)
	spew.Dump(schema)
	//fmt.Printf("%s", schema)
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: "pulsar://localhost:6650",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	chainName := "localterra"
	eventKind := "tm.event='NewBlock'"
	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            "persistent://terra/" + chainName + "/" + eventKind,
		SubscriptionName: "my-sub2",
		Type:             pulsar.Exclusive,
		//	Schema:                      consumerJS,
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

	fmt.Printf("Received message msgId: %#v -- content: '%s'\n",
		msg.ID(), string(msg.Payload()))
}
