package producer

import (
	"context"
	"fmt"
	"log"

	"github.com/apache/pulsar-client-go/pulsar"
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
func SendMessage(producer Instance, log *zap.SugaredLogger, message pulsar.ProducerMessage) {

	msg_id, err := producer.p.Send(producer.ctx, &message)
	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	log.Debugw("submitted message in:", "data.Query", "with", "MessageId", fmt.Sprint(msg_id.EntryID()))

}
