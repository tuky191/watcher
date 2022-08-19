package producer

import (
	"context"
	"fmt"
	"log"

	"encoding/json"

	"github.com/apache/pulsar-client-go/pulsar"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"go.uber.org/zap"
)

type Instance struct {
	client pulsar.Client
	p      pulsar.Producer
	ctx    context.Context
	o      Options
}

type Options struct {
	ProducerOptions pulsar.ProducerOptions
	ClientOptions   pulsar.ClientOptions
}

func New(o *Options) (*Instance, error) {
	client, err := pulsar.NewClient(o.ClientOptions)
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
		return nil, err
	}

	producer, err := NewWithClient(client, &o.ProducerOptions)
	if err != nil {
		log.Fatalf("Could not start the producer: %v", err)
		return nil, err
	}
	ii := &Instance{
		client: client,
		p:      producer,
		o:      *o,
	}

	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	return ii, nil
}

func NewWithClient(c pulsar.Client, p *pulsar.ProducerOptions) (pulsar.Producer, error) {
	producer, err := c.CreateProducer(pulsar.ProducerOptions{
		Topic: p.Topic,
	})
	if err != nil {
		log.Fatalf("Could not start the producer: %v", err)
		return nil, err
	}
	return producer, err
}
func SendMessage(producers map[string]Instance, log *zap.SugaredLogger, data coretypes.ResultEvent) {
	b, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	msg_id, err := producers[data.Query].p.Send(producers[data.Query].ctx, &pulsar.ProducerMessage{
		Payload: []byte(string(b)),
		//SequenceID: &data.Block.Height,
	})
	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	log.Debugw("submitted message in:", data.Query, "with", "MessageId", fmt.Sprint(msg_id.EntryID()))

}
