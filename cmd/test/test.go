package main

import (
	"context"
	"fmt"
	"log"
	"rpc_watcher/rpcwatcher/avro"
	"rpc_watcher/rpcwatcher/logging"
	producer "rpc_watcher/rpcwatcher/pulsar"
	"rpc_watcher/rpcwatcher/sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	pulsar_logger "github.com/apache/pulsar-client-go/pulsar/log"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	block_feed "github.com/terra-money/mantlemint/block_feed"
)

type testJSON struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Custom string `json:"custom"`
}

var (
	exampleSchemaDef = "{\"type\":\"record\",\"name\":\"Example\",\"namespace\":\"test\"," +
		"\"fields\":[{\"name\":\"ID\",\"type\":\"int\"},{\"name\":\"Name\",\"type\":\"string\"}]}"
	protoSchemaDef = "{\"type\":\"record\",\"name\":\"Example\",\"namespace\":\"test\"," +
		"\"fields\":[{\"name\":\"num\",\"type\":\"int\"},{\"name\":\"msf\",\"type\":\"string\"}]}"
)

func main() {
	l := logging.New(logging.LoggingConfig{
		Debug: true,
		JSON:  true,
	})
	sync_instance := sync.New("http://127.0.0.1:26657", l)
	block, err := sync.GetBlock(1, sync_instance)
	if err != nil {
		l.Errorw("Unable to get block", "url_string", "error", err)
	}
	//spew.Dump(block)
	//log.Fatal()
	//#####################################################################################################
	b := &block_feed.BlockResult{}
	schema := avro.GenerateAvroSchema(&block_feed.BlockResult{})
	properties := make(map[string]string)
	properties["pulsar"] = "EHLO"
	jsonSchemaWithProperties := pulsar.NewJSONSchema(schema, properties)

	// schemaPayload, err := jsonSchemaWithProperties.Encode(block)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//spew.Dump(schemaPayload)
	//err = jsonSchemaWithProperties.Decode(schemaPayload, b)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println("#########################################################")
	// spew.Dump(b)
	// fmt.Println("#########################################################")

	logrus_logger := logrus.StandardLogger()
	logrus_logger.SetLevel(logrus.InfoLevel)
	o := producer.Options{
		ClientOptions: pulsar.ClientOptions{
			URL:               "pulsar://localhost:6650",
			OperationTimeout:  30 * time.Second,
			ConnectionTimeout: 30 * time.Second,
			Logger:            pulsar_logger.NewLoggerWithLogrus(logrus_logger),
		},
		ProducerOptions: pulsar.ProducerOptions{Topic: "persistent://terra/localterra/tm.event='NewBlock'", Schema: jsonSchemaWithProperties},
	}
	//spew.Dump(o.ProducerOptions.Schema)
	p, err := producer.New(&o)
	if err != nil {
		fmt.Printf("%s", err)
	}
	message := pulsar.ProducerMessage{
		Value: block,
	}

	producer.SendMessage(*p, l, message)

	consumerJS := pulsar.NewJSONSchema(schema, nil)

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: "pulsar://localhost:6650",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:                       "persistent://terra/localterra/tm.event='NewBlock'",
		SubscriptionName:            "my-sub3",
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
	//fmt.Println(s.ID)
	spew.Dump(b.BlockID)
	spew.Dump(b.Block)
	fmt.Printf("Received message msgId: %#v\n",
		msg.ID())
	consumer.Ack(msg)

}
