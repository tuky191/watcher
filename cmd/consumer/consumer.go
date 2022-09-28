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

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: "pulsar://localhost:6650",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	b := &block_feed.BlockResult{}
	schema, err := avro.GenerateAvroSchema(&block_feed.BlockResult{})

	if err != nil {
		log.Fatal(err)
	}
	consumerJS := pulsar.NewJSONSchema(schema, nil)

	chainName := "localterra"
	eventKind := "newblock"
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
	for {
		msg, err := consumer.Receive(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		err = msg.GetSchemaValue(&b)
		if err != nil {
			log.Fatal(err)
		}

		spew.Dump(b)
		fmt.Printf("Received message msgId: %#v -- content:\n '%+v'\n",
			msg.ID(), b)
		consumer.Ack(msg)
	}

	// tx := types.EventDataTx{}
	// schema_tx, err := avro.GenerateAvroSchema(abci.TxResult{})

	// consumerTx := pulsar.NewJSONSchema(schema_tx, nil)

	// chainName := "localterra"
	// eventKind := "tx"
	// consumer, err := client.Subscribe(pulsar.ConsumerOptions{
	// 	Topic:                       "persistent://terra/" + chainName + "/" + eventKind,
	// 	SubscriptionName:            "my-sub1",
	// 	Type:                        pulsar.Exclusive,
	// 	Schema:                      consumerTx,
	// 	SubscriptionInitialPosition: pulsar.SubscriptionPositionEarliest,
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// defer consumer.Close()

	// msg, err := consumer.Receive(context.Background())
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// err = msg.GetSchemaValue(&tx)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// marsh_json, err := json.Marshal(tx)
	// fmt.Printf("%s", marsh_json)

	// fmt.Printf("Received message msgId: %#v -- content:\n '%+v'\n",
	// 	msg.ID(), tx)

}
