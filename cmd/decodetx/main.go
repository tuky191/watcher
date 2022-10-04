package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"rpc_watcher/rpcwatcher/avro"
	"rpc_watcher/rpcwatcher/logging"
	"strconv"
	"time"

	log "github.com/apache/pulsar/pulsar-function-go/logutil"

	pulsar_types "rpc_watcher/rpcwatcher/helper/types/pulsar"
	pulsar_producer "rpc_watcher/rpcwatcher/pulsar"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/apache/pulsar/pulsar-function-go/pf"
	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tendermint "github.com/tendermint/tendermint/types"
	terra "github.com/terra-money/core/v2/app"
)

type TxRecord struct {
	Code      uint32          `json:"code"`
	Codespace string          `json:"codespace"`
	GasUsed   int64           `json:"gas_used"`
	GasWanted int64           `json:"gas_wanted"`
	Height    int64           `json:"height"`
	RawLog    string          `json:"raw_log"`
	Logs      json.RawMessage `json:"logs"`
	TxHash    string          `json:"txhash"`
	Timestamp time.Time       `json:"timestamp"`
	Tx        json.RawMessage `json:"tx"`
	Events    []abci.Event    `json:"events"`
}

var cdc = terra.MakeEncodingConfig()
var schema, _ = avro.GenerateAvroSchema(&TxRecord{})
var topic string = ""
var p pulsar_types.Producer

func decodeTx(txResult abci.TxResult) (TxRecord, error) {
	var txByte tendermint.Tx = txResult.Tx
	decoded_tx := TxRecord{}
	txDecoder := cdc.TxConfig.TxDecoder()
	jsonEncoder := cdc.TxConfig.TxJSONEncoder()

	tx, decodeErr := txDecoder(txByte)
	if decodeErr != nil {
		return decoded_tx, decodeErr
	}
	hash := txByte.Hash()
	txJSON, _ := jsonEncoder(tx)

	decoded_tx.TxHash = fmt.Sprintf("%X", hash)
	decoded_tx.Code = txResult.Result.Code
	decoded_tx.Codespace = txResult.Result.Codespace
	decoded_tx.GasUsed = txResult.Result.GasUsed
	decoded_tx.GasWanted = txResult.Result.GasWanted
	decoded_tx.Height = txResult.Height
	decoded_tx.RawLog = txResult.Result.Log
	decoded_tx.Events = txResult.Result.Events
	decoded_tx.Logs = func() json.RawMessage {
		if txResult.Result.Code == 0 {
			return []byte(txResult.Result.Log)
		} else {
			out, _ := json.Marshal([]string{})
			return out
		}
	}()

	decoded_tx.Tx = txJSON
	return decoded_tx, nil
}

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
	decodedTx, err := decodeTx(*tx)
	decodedTx.Timestamp = record.EventTime()
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
				URL:               "pulsar://localhost:6650",
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
