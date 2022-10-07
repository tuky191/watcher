package main

import (
	"context"
	"errors"
	"fmt"
	"rpc_watcher/rpcwatcher/avro"
	"strconv"

	log "github.com/apache/pulsar/pulsar-function-go/logutil"

	rpc_types "rpc_watcher/rpcwatcher/helper/types/rpc"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/apache/pulsar/pulsar-function-go/pf"
	abci "github.com/tendermint/tendermint/abci/types"

	"rpc_watcher/rpcwatcher/helper/rpc"
)

var topic string = ""
var schema, _ = avro.GenerateAvroSchema(&rpc_types.TxRecord{})
var jsonSchemaWithProperties *pulsar.JSONSchema
var properties = make(map[string]string)
var producer pulsar.Producer

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
	if record.EventTime().Unix() > 0 {
		decodedTx.Timestamp = record.EventTime()
		log.Infof("Timestamp set to message EventTime: %s", record.EventTime().String())
	} else {
		decodedTx.Timestamp = record.PublishTime()
		log.Infof("EventTime not set, Timestamp set to message PublishTime: %s", record.PublishTime().String())
	}

	if err != nil {
		log.Fatalf("Unable to decode source tx:", "tx", record.ID(), "error:", err)
	}

	if err != nil {
		log.Fatal(err)
	}

	if topic == "" {
		topic = fctx.GetOutputTopic()
		properties["pulsar"] = "EHLO"
		jsonSchemaWithProperties = pulsar.NewJSONSchema(schema, properties)
		producer = fctx.NewOutputMessage(fctx.GetOutputTopic())
	}

	message := pulsar.ProducerMessage{
		Value:       decodedTx,
		SequenceID:  &decodedTx.Height,
		OrderingKey: strconv.FormatInt(decodedTx.Height, 10),
		EventTime:   decodedTx.Timestamp,
		Schema:      jsonSchemaWithProperties,
	}

	msg, err := producer.Send(ctx, &message)
	if err != nil {
		log.Fatalf("Unable to send message", "error:", err)
	}

	log.Infof("Submitted tx: %s at height: %s with timestamp: %s as (%s,%s,%s)", decodedTx.TxHash, fmt.Sprint(decodedTx.Height), decodedTx.Timestamp.String(), fmt.Sprint(msg.LedgerID()), fmt.Sprint(msg.EntryID(), fmt.Sprint(msg.PartitionIdx())))

	return nil
}

func main() {
	pf.Start(PublishFunc)
}
