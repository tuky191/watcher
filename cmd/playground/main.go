package main

import (
	"context"
	"fmt"
	"log"
	"rpc_watcher/rpcwatcher/avro"

	"github.com/apache/pulsar-client-go/pulsar"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	// l := logging.New(logging.LoggingConfig{
	// 	Debug: true,
	// 	JSON:  true,
	// })

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
	//schema, _ := avro.GenerateAvroSchema(&block_feed.BlockResult{})

	schema, _ := avro.GenerateAvroSchema(abci.TxResult{})
	//spew.Dump(schema)
	// c, err := rpcwatcher.ReadConfig()
	// if err != nil {
	// 	panic(err)
	// }
	// //sync_instance := sync.New(c, l)
	// rpc_instance := rpc.NewRPCApi(c, l)

	// txs := rpc_instance.GetTxsFromBlockByHeight(int64(42961))
	// //txs := rpc_instance.GetTxsFromBlockByHeight(int64(1763383))
	// spew.Dump(txs)
	// decoded_tx, err := rpc.DecodeTx(txs[0])
	// if err != nil {
	// 	l.Errorw("Unable to decode tx", "error", err)
	// }
	// json_txt, err := tmjson.Marshal(decoded_tx)
	// fmt.Printf("%s\n", json_txt)
	// spew.Dump(json_txt)
	// log.Fatal()
	// block, err := sync_instance.GetBlockByHeight(1)
	// if err != nil {
	// 	l.Errorw("Unable to get block", "url_string", "error", err)
	// }
	// spew.Dump(block)
	// block, err := sync_instance.GetLatestBlock()
	// if err != nil {
	// 	l.Errorw("Unable to get block", "url_string", "error", err)
	// }
	// spew.Dump(block)
	// log.Fatal()

	// 	b := &block_feed.BlockResult{}
	properties := make(map[string]string)
	properties["pulsar"] = "EHLO"
	jsonSchemaWithProperties := pulsar.NewJSONSchema(schema, properties)

	// 	logrus_logger := logrus.StandardLogger()
	// 	logrus_logger.SetLevel(logrus.InfoLevel)
	// 	o := producer.PulsarOptions{
	// 		ClientOptions: pulsar.ClientOptions{
	// 			URL:               "pulsar://localhost:6650",
	// 			OperationTimeout:  30 * time.Second,
	// 			ConnectionTimeout: 30 * time.Second,
	// 			Logger:            pulsar_logger.NewLoggerWithLogrus(logrus_logger),
	// 		},
	// 		ProducerOptions: pulsar.ProducerOptions{Topic: "persistent://terra/localterra/tm.event='NewBlock'", Schema: jsonSchemaWithProperties},
	// 		TableViewOptions: pulsar.TableViewOptions{
	// 			Topic: "persistent://terra/localterra/tm.event='NewBlock'", Schema: jsonSchemaWithProperties, SchemaValueType: reflect.TypeOf(&block_feed.BlockResult{}),
	// 		},
	// 	}

	// 	//table_view_config :=
	// 	table_view, err := producer.NewTableView(&o)
	// 	if err != nil {
	// 		fmt.Printf("%s", err)
	// 	}
	// 	table_view.GetElement(l)
	// 	log.Fatal("Lst message received")
	// 	p, err := producer.NewProducer(&o)
	// 	if err != nil {
	// 		fmt.Printf("%s", err)
	// 	}
	// 	message := pulsar.ProducerMessage{
	// 		Value: block,
	// 	}

	// 	p.SendMessage(l, message)

	//consumerJS := pulsar.NewJSONSchema(schema, nil)

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: "pulsar://localhost:6650",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:                       "persistent://terra/pisco-1/tx",
		SubscriptionName:            "my-sub3",
		Type:                        pulsar.Exclusive,
		Schema:                      jsonSchemaWithProperties,
		SubscriptionInitialPosition: pulsar.SubscriptionPositionEarliest,
	})
	msg_id := pulsar.NewMessageID(305, 806, -1, 0)

	consumer.Seek(msg_id)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()
	b := abci.TxResult{}
	msg, err := consumer.Receive(context.Background())
	//spew.Dump(msg)
	//spew.Dump(msg.EventTime().Unix())
	if msg.EventTime().Unix() < 0 {
		fmt.Printf("smaller than 0")
		spew.Dump(msg.EventTime().Unix())
	} else {
		fmt.Printf("bigger than 0")
		spew.Dump(msg.EventTime().Unix())
	}
	if err != nil {
		log.Fatal(err)
	}
	err = msg.GetSchemaValue(&b)
	if err != nil {
		log.Fatal(err)
	}
	//spew.Dump(msg.EventTime())
	// fmt.Printf("Received message msgId: %#v\n with\n block id: %s block: %s",
	// 	msg.ID(), spew.Sdump(b.BlockID), spew.Sdump(b.Block))

	//consumer.Ack(msg)

}
