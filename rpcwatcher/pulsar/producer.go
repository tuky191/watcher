package pulsar

import (
	"log"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

type ClientOptions struct {
	URL               string        `validate:"required"`
	OperationTimeout  time.Duration `validate:"required"`
	ConnectionTimeout time.Duration `validate:"required"`
}

func New(c *ClientOptions) (pulsar.Client, error) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               c.URL,
		OperationTimeout:  c.OperationTimeout,
		ConnectionTimeout: c.ConnectionTimeout,
	})
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
	}

	defer client.Close()
	return client, err
}
