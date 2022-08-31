package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"rpc_watcher/rpcwatcher/avro"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/davecgh/go-spew/spew"
	"github.com/terra-money/mantlemint/block_feed"
)

func InspectStructV(val reflect.Value) {
	if val.Kind() == reflect.Interface && !val.IsNil() {
		elm := val.Elem()
		if elm.Kind() == reflect.Ptr && !elm.IsNil() && elm.Elem().Kind() == reflect.Ptr {
			val = elm
		}
	}
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		address := "not-addressable"

		if valueField.Kind() == reflect.Interface && !valueField.IsNil() {
			elm := valueField.Elem()
			if elm.Kind() == reflect.Ptr && !elm.IsNil() && elm.Elem().Kind() == reflect.Ptr {
				valueField = elm
			}
		}

		if valueField.Kind() == reflect.Ptr {
			valueField = valueField.Elem()

		}
		if valueField.CanAddr() {
			address = fmt.Sprintf("0x%X", valueField.Addr().Pointer())
		}

		fmt.Printf("Field Name: %s,\t Address: %v\t, Field type: %v\t, Field kind: %v\n", typeField.Name,
			address, typeField.Type, valueField.Kind())

		if valueField.Kind() == reflect.Struct {
			InspectStructV(valueField)
		}
	}
}

func InspectStruct(v interface{}) {
	InspectStructV(reflect.ValueOf(v))
}

func main() {

	//v := reflect.ValueOf(block_feed.BlockResult{})

	//InspectStruct(tendermint.Block{})
	//log.Fatal()
	avro_schema := avro.GenerateAvroSchema(block_feed.BlockResult{})
	spew.Dump(avro_schema)
	log.Fatal()

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: "pulsar://localhost:6650",
	})

	defer client.Close()

	chainName := "localterra"
	eventKind := "tm.event='NewBlock'"
	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            "persistent://terra/" + chainName + "/" + eventKind,
		SubscriptionName: "my-sub1",
		Type:             pulsar.Exclusive,
	})

	defer consumer.Close()

	msg, err := consumer.Receive(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Received message msgId: %#v -- content: '%s'\n",
		msg.ID(), string(msg.Payload()))
}
