package producer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/davecgh/go-spew/spew"
)

type Instance struct {
	p   pulsar.Producer
	ctx context.Context
}

type ClientOptions struct {
	URL               string        `validate:"required"`
	OperationTimeout  time.Duration `validate:"required"`
	ConnectionTimeout time.Duration `validate:"required"`
}

type ProducerOptions struct {
	Topic                   string `validate:"required"`
	Name                    string
	Properties              map[string]string
	SendTimeout             time.Duration
	DisableBlockIfQueueFull bool
	MaxPendingMessages      int
	DisableBatching         bool
	BatchingMaxPublishDelay time.Duration

	BatchingMaxMessages uint
	BatchingMaxSize     uint
}

func New(co *ClientOptions, p *ProducerOptions) (*Instance, error) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               co.URL,
		OperationTimeout:  co.OperationTimeout,
		ConnectionTimeout: co.ConnectionTimeout,
	})
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
		return nil, err
	}

	//defer client.Close()
	producer, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic: p.Topic,
	})

	//producer, err := NewWithClient(client, p)
	if err != nil {
		log.Fatalf("Could not start the producer: %v", err)
		return nil, err
	}
	ii := &Instance{
		p: producer,
	}
	//defer producer.Close()

	ds, err := ii.p.Send(ii.ctx, &pulsar.ProducerMessage{
		Payload: []byte("hello"),
		//	SequenceID: &realData.Block.Height,
	})
	spew.Dump(ds)
	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	return ii, nil
}

/*
	func NewWithClient(c pulsar.Client, p *ProducerOptions) (*Instance, error) {
		producer, err := c.CreateProducer(pulsar.ProducerOptions{
			Topic: p.Topic,
		})
		if err != nil {
			log.Fatalf("Could not start the producer: %v", err)
			return nil, err
		}
		ii := &Instance{
			p: producer,
		}
		defer producer.Close()
		return ii, err
	}
*/
func SendMessage(p *Instance) {
	_, err := p.p.Send(p.ctx, &pulsar.ProducerMessage{
		Payload: []byte("hello"),
		//	SequenceID: &realData.Block.Height,
	})
	if err != nil {
		fmt.Println("Failed to publish message", err)
	}

}
