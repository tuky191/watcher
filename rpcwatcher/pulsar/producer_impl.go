package pulsar

import (
	"context"
	"fmt"
	"log"
	pulsar_types "rpc_watcher/rpcwatcher/helper/types/pulsar"

	"github.com/apache/pulsar-client-go/pulsar"
	"go.uber.org/zap"
)

type ProducerInstance struct {
	client   pulsar.Client
	Producer pulsar.Producer
	ctx      context.Context
	o        pulsar_types.PulsarOptions
}

func NewProducer(o *pulsar_types.PulsarOptions) (pulsar_types.Producer, error) {
	client, err := pulsar.NewClient(o.ClientOptions)
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
		return nil, err
	}

	producer, err := NewProducerWithClient(client, o.ProducerOptions)
	if err != nil {
		log.Fatalf("Could not start the producer: %v", err)
		return nil, err
	}
	ii := &ProducerInstance{
		client:   client,
		Producer: producer,
		o:        *o,
	}

	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	return ii, nil
}

func NewProducerWithClient(c pulsar.Client, p pulsar.ProducerOptions) (pulsar.Producer, error) {
	producer, err := c.CreateProducer(p)
	if err != nil {
		log.Fatalf("Could not start the producer: %v", err)
		return nil, err
	}
	return producer, err
}
func (i *ProducerInstance) SendMessage(log *zap.SugaredLogger, message pulsar.ProducerMessage) {
	msg_id, err := i.Producer.Send(i.ctx, &message)
	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	log.Debugw("submitted message in:", i.Producer.Topic(), "with", "MessageId", fmt.Sprint(msg_id.EntryID()))

}
