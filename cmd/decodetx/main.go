package main

import (
	"context"
	"errors"
	"rpc_watcher/rpcwatcher/avro"
	"rpc_watcher/rpcwatcher/logging"
	"strconv"
	"time"

	log "github.com/apache/pulsar/pulsar-function-go/logutil"

	pulsar_types "rpc_watcher/rpcwatcher/helper/types/pulsar"
	pulsar_producer "rpc_watcher/rpcwatcher/pulsar"

	rpc_types "rpc_watcher/rpcwatcher/helper/types/rpc"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/apache/pulsar/pulsar-function-go/pf"
	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"

	"rpc_watcher/rpcwatcher/helper/rpc"
)

var topic string = ""
var schema, _ = avro.GenerateAvroSchema(&rpc_types.TxRecord{})
var p pulsar_types.Producer

func PublishFunc(ctx context.Context, in []byte) error {
	fctx, ok := pf.FromContext(ctx)
	tx := &abci.TxResult{}

	if !ok {
		return errors.New("get Go Functions Context error")
	}
	record := fctx.GetCurrentRecord()
	err := record.GetSchemaValue(&tx)

	if err != nil {
		log.Fatalf("Unable to unmarshall source tx:", "tx", record.ID(), "error:", err)
	}

	decodedTx, err := rpc.DecodeTx(*tx)

	//Temp fix for txs that were retrieved via websocket and didnt have event_time set. Already rectified by setting the event_time from the block timestamp value.
	if !record.EventTime().IsZero() {
		decodedTx.Timestamp = record.EventTime()
	} else {
		decodedTx.Timestamp = record.PublishTime()

	}

	if err != nil {
		log.Fatalf("Unable to decode source tx:", "tx", record.ID(), "error:", err)
	}

	decodedTxjson, err := tmjson.Marshal(decodedTx)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("%s", decodedTxjson)
	if topic == "" {
		topic = "persistent://" + fctx.GetTenantAndNamespace() + "/" + "txdecoded"
		properties := make(map[string]string)
		properties["pulsar"] = "EHLO"
		jsonSchemaWithProperties := pulsar.NewJSONSchema(schema, properties)
		o := pulsar_types.PulsarOptions{
			ClientOptions: pulsar.ClientOptions{
				URL:               "pulsar://proxy1:6650",
				OperationTimeout:  30 * time.Second,
				ConnectionTimeout: 30 * time.Second,
			},
			ProducerOptions: pulsar.ProducerOptions{Topic: topic, Schema: jsonSchemaWithProperties},
		}
		p, err = pulsar_producer.NewProducer(&o)
		if err != nil {
			log.Fatalf("Unable to start producer", "error:", err)
		}
	}

	message := pulsar.ProducerMessage{
		Value:       decodedTx,
		SequenceID:  &decodedTx.Height,
		OrderingKey: strconv.FormatInt(decodedTx.Height, 10),
		EventTime:   decodedTx.Timestamp,
	}
	l := logging.New(logging.LoggingConfig{
		Debug: true,
		JSON:  true,
	})

	p.SendMessage(l, message)
	return nil
}

func main() {
	pf.Start(PublishFunc)
}
