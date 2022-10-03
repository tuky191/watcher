package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"rpc_watcher/rpcwatcher"
	"rpc_watcher/rpcwatcher/logging"
	"rpc_watcher/rpcwatcher/sync"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/davecgh/go-spew/spew"
	tendermint "github.com/tendermint/tendermint/types"
	mantlemint_types "github.com/terra-money/mantlemint/indexer/tx"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/apache/pulsar/pulsar-function-go/pf"
	terra "github.com/terra-money/core/v2/app"
)

type TxRecord struct {
	Tx         json.RawMessage `json:"tx"`
	TxResponse json.RawMessage `json:"tx_response"`
}

var cdc = terra.MakeEncodingConfig()

func decodeTx(txByte tendermint.Tx) (types.Tx, error) {
	// encoder; proto -> mem -> json
	decoded_tx := mantlemint_types.TxByHeightRecord{}
	spew.Dump(decoded_tx)
	txDecoder := cdc.TxConfig.TxDecoder()
	jsonEncoder := cdc.TxConfig.TxJSONEncoder()

	hash := txByte.Hash()
	tx, decodeErr := txDecoder(txByte)
	spew.Dump(tx)
	if decodeErr != nil {
		return nil, decodeErr
	}
	enc, err := jsonEncoder(tx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s", string(enc))
	//spew.Dump()
	decoded_tx.TxHash = fmt.Sprintf("%X", hash)
	spew.Dump(decoded_tx)
	return tx, nil

}

func PublishFunc(ctx context.Context, in []byte) error {
	fctx, ok := pf.FromContext(ctx)
	if !ok {
		return errors.New("get Go Functions Context error")
	}

	publishTopic := "publish-topic"
	output := append(in, 110)

	producer := fctx.NewOutputMessage(publishTopic)
	msgID, err := producer.Send(ctx, &pulsar.ProducerMessage{
		Payload: output,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("The output message ID is: %+v", msgID)
	return nil
}

func main() {
	c, err := rpcwatcher.ReadConfig()
	if err != nil {
		panic(err)
	}
	l := logging.New(logging.LoggingConfig{
		Debug: c.Debug,
		JSON:  c.JSONLogs,
	})

	sync := sync.New(c, l)
	height := int64(1081959)
	TxResults := sync.GetTxsFromBlockByHeight(height)
	tx := TxResults[1].Tx
	//TxResults[1].Result
	spew.Dump(TxResults[1].Result)

	decodedTx, err := decodeTx(tx)
	fmt.Printf("%s", decodedTx)
	// output, err := tmjson.Marshal(decodedTx)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// spew.Dump(output)
	//pf.Start(PublishFunc)
}
