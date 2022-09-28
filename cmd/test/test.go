package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"rpc_watcher/rpcwatcher"
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

func main() {
	l := logging.New(logging.LoggingConfig{
		Debug: true,
		JSON:  true,
	})

	// db_config := &presto.Config{
	// 	PrestoURI: "http://root@localhost:8082",
	// }
	// db, err := database.New(db_config)
	// if err != nil {
	// 	l.Errorw("Unable to create presto db handle", "error", err)
	// }
	// min := "1000"
	// max := "2000"
	// result, err := db.Handle.Query(`select __sequence_id__ from pulsar."terra/localterra"."tm.event='newblock'" where __sequence_id__ > ` + min + ` and __sequence_id__ < ` + max)

	// if err != nil {
	// 	l.Errorw("Unable query pulsar", "error", err)
	// }
	// for result.Next() {
	// 	var height int64
	// 	if err := result.Scan(&height); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Printf("height is %d\n", height)
	// }
	// if err := result.Err(); err != nil {
	// 	log.Fatal(err)
	// }
	//fmt.Println(result)
	//spew.Dump(result)

	//log.Fatal()
	schema, _ := avro.GenerateAvroSchema(&block_feed.BlockResult{})

	//schema := avro.GenerateAvroSchema(types.EventDataTx{})
	//spew.Dump(schema)
	c, err := rpcwatcher.ReadConfig()
	if err != nil {
		panic(err)
	}
	sync_instance := sync.New(c, l)
	latest, _ := sync_instance.GetLatestPublishedBlock()
	if latest.Block == nil {
		spew.Dump(latest.Block)
	}

	log.Fatal()
	// block, err := sync_instance.GetBlockByHeight(1)
	// if err != nil {
	// 	l.Errorw("Unable to get block", "url_string", "error", err)
	// }
	// spew.Dump(block)
	block, err := sync_instance.GetLatestBlock()
	if err != nil {
		l.Errorw("Unable to get block", "url_string", "error", err)
	}
	// spew.Dump(block)
	// log.Fatal()

	b := &block_feed.BlockResult{}
	properties := make(map[string]string)
	properties["pulsar"] = "EHLO"
	jsonSchemaWithProperties := pulsar.NewJSONSchema(schema, properties)

	logrus_logger := logrus.StandardLogger()
	logrus_logger.SetLevel(logrus.InfoLevel)
	o := producer.PulsarOptions{
		ClientOptions: pulsar.ClientOptions{
			URL:               "pulsar://localhost:6650",
			OperationTimeout:  30 * time.Second,
			ConnectionTimeout: 30 * time.Second,
			Logger:            pulsar_logger.NewLoggerWithLogrus(logrus_logger),
		},
		ProducerOptions: pulsar.ProducerOptions{Topic: "persistent://terra/localterra/tm.event='NewBlock'", Schema: jsonSchemaWithProperties},
		TableViewOptions: pulsar.TableViewOptions{
			Topic: "persistent://terra/localterra/tm.event='NewBlock'", Schema: jsonSchemaWithProperties, SchemaValueType: reflect.TypeOf(&block_feed.BlockResult{}),
		},
	}

	//table_view_config :=
	table_view, err := producer.NewTableView(&o)
	if err != nil {
		fmt.Printf("%s", err)
	}
	table_view.GetElement(l)
	log.Fatal("Lst message received")
	p, err := producer.NewProducer(&o)
	if err != nil {
		fmt.Printf("%s", err)
	}
	message := pulsar.ProducerMessage{
		Value: block,
	}

	p.SendMessage(l, message)

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
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Received message msgId: %#v\n with\n block id: %s block: %s",
		msg.ID(), spew.Sdump(b.BlockID), spew.Sdump(b.Block))

	consumer.Ack(msg)

}
