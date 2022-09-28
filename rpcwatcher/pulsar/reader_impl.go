package pulsar

import (
	"context"
	"fmt"
	"log"

	"github.com/apache/pulsar-client-go/pulsar"
	"go.uber.org/zap"
)

type ReaderInstance struct {
	client pulsar.Client
	Reader pulsar.Reader
	o      PulsarOptions
}

func NewReader(o *PulsarOptions) (Reader, error) {
	client, err := pulsar.NewClient(o.ClientOptions)
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
		return nil, err
	}

	reader, err := NewReaderWithClient(client, o.ReaderOptions)
	if err != nil {
		log.Fatalf("Could not create reader: %v", err)
		return nil, err
	}
	ii := &ReaderInstance{
		client: client,
		Reader: reader,
		o:      *o,
	}

	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	return ii, nil
}

func NewReaderWithClient(c pulsar.Client, p pulsar.ReaderOptions) (pulsar.Reader, error) {
	reader, err := c.CreateReader(p)
	if err != nil {
		log.Fatalf("Could not create reader: %v", err)
		return nil, err
	}
	return reader, err
}

func (i *ReaderInstance) ReadLastMessage(log *zap.SugaredLogger) (pulsar.Message, error) {
	var msg pulsar.Message
	var err error
	for i.Reader.HasNext() {
		msg, err = i.Reader.Next(context.Background())
		if err != nil {
			log.Errorw("cannot read latest message", "error", err)

		}
		if err != nil {
			return nil, err
		}
		log.Debugw("received message with", "MessageId", fmt.Sprint(msg.ID()))
	}
	return msg, nil
}
