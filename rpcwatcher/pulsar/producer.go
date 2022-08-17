package producer

import (
	"context"
	"fmt"
	"log"
	"time"

	"encoding/json"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/tendermint/tendermint/types"
	"go.uber.org/zap"
)

type Instance struct {
	client pulsar.Client
	p      pulsar.Producer
	ctx    context.Context
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

	producer, err := NewWithClient(client, p)
	if err != nil {
		log.Fatalf("Could not start the producer: %v", err)
		return nil, err
	}
	ii := &Instance{
		client: client,
		p:      producer,
	}

	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	return ii, nil
}

func NewWithClient(c pulsar.Client, p *ProducerOptions) (pulsar.Producer, error) {
	producer, err := c.CreateProducer(pulsar.ProducerOptions{
		Topic: p.Topic,
	})
	if err != nil {
		log.Fatalf("Could not start the producer: %v", err)
		return nil, err
	}
	return producer, err
}
func SendMessage(p *Instance, log *zap.SugaredLogger, data types.EventDataNewBlock) {
	b, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return
	}
	msg_id, err := p.p.Send(p.ctx, &pulsar.ProducerMessage{
		Payload:    []byte(string(b)),
		SequenceID: &data.Block.Height,
	})
	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	log.Debugw("submitted message with", "MessageId", fmt.Sprint(msg_id.EntryID()))

}
