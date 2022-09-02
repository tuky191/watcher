package main

import (
	"context"
	"fmt"
	"log"
	"rpc_watcher/rpcwatcher/logging"
	producer "rpc_watcher/rpcwatcher/pulsar"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/davecgh/go-spew/spew"
)

type testJSON struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var (
	exampleSchemaDef = "{\"type\":\"record\",\"name\":\"Example\",\"namespace\":\"test\"," +
		"\"fields\":[{\"name\":\"ID\",\"type\":\"int\"},{\"name\":\"Name\",\"type\":\"string\"}]}"
	protoSchemaDef = "{\"type\":\"record\",\"name\":\"Example\",\"namespace\":\"test\"," +
		"\"fields\":[{\"name\":\"num\",\"type\":\"int\"},{\"name\":\"msf\",\"type\":\"string\"}]}"
)

func main() {

	// schema := avro.GenerateAvroSchema(&block_feed.BlockResult{})
	// properties := make(map[string]string)
	// jsonSchemaWithProperties := pulsar.NewJSONSchema(schema, properties)
	// spew.Dump(jsonSchemaWithProperties)

	properties := make(map[string]string)
	properties["pulsar"] = "hello"
	jsonSchemaWithProperties := pulsar.NewJSONSchema(exampleSchemaDef, properties)

	o := producer.Options{
		ClientOptions: pulsar.ClientOptions{
			URL:               "pulsar://localhost:6650",
			OperationTimeout:  30 * time.Second,
			ConnectionTimeout: 30 * time.Second,
		},
		ProducerOptions: pulsar.ProducerOptions{Topic: "persistent://terra/localterra/test", Schema: jsonSchemaWithProperties},
	}
	p, err := producer.New(&o)
	message := pulsar.ProducerMessage{
		Value: &testJSON{
			ID:   100,
			Name: "pulsar",
		},
		//Payload:     []byte(string(b)),
	}
	l := logging.New(logging.LoggingConfig{
		Debug: true,
		JSON:  true,
	})
	producer.SendMessage(*p, l, message)

	//consumerJS := pulsar.NewJSONSchema(schema, nil)
	//spew.Dump(schema)
	//fmt.Printf("%s", schema)
	//log.Fatal()

	var s testJSON

	consumerJS := pulsar.NewJSONSchema(exampleSchemaDef, nil)

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: "pulsar://localhost:6650",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:                       "persistent://terra/localterra/test",
		SubscriptionName:            "my-sub2",
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
	err = msg.GetSchemaValue(&s)
	spew.Dump(&s)

	fmt.Printf("Received message msgId: %#v -- content: '%s'\n",
		msg.ID(), string(msg.Payload()))
}
