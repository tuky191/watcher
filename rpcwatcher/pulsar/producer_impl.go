package pulsar

import (
	"context"
	"fmt"
	"log"

	"github.com/apache/pulsar-client-go/pulsar"
	"go.uber.org/zap"
)

type Instance struct {
	client   pulsar.Client
	Producer pulsar.Producer
	ctx      context.Context
	o        PulsarOptions
}

func New(o *PulsarOptions) (Producer, error) {
	client, err := pulsar.NewClient(o.ClientOptions)
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
		return nil, err
	}

	producer, err := NewWithClient(client, o.ProducerOptions)
	if err != nil {
		log.Fatalf("Could not start the producer: %v", err)
		return nil, err
	}
	ii := &Instance{
		client:   client,
		Producer: producer,
		o:        *o,
	}

	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	return ii, nil
}

func NewWithClient(c pulsar.Client, p pulsar.ProducerOptions) (pulsar.Producer, error) {
	producer, err := c.CreateProducer(p)
	if err != nil {
		log.Fatalf("Could not start the producer: %v", err)
		return nil, err
	}
	return producer, err
}
func (i *Instance) SendMessage(log *zap.SugaredLogger, message pulsar.ProducerMessage) {

	msg_id, err := i.Producer.Send(i.ctx, &message)
	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	log.Debugw("submitted message in:", "data.Query", "with", "MessageId", fmt.Sprint(msg_id.EntryID()))

}
