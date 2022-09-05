package pulsar

import (
	"github.com/apache/pulsar-client-go/pulsar"
	"go.uber.org/zap"
)

type PulsarOptions struct {
	ProducerOptions  pulsar.ProducerOptions
	ClientOptions    pulsar.ClientOptions
	TableViewOptions pulsar.TableViewOptions
}

type Producer interface {
	SendMessage(log *zap.SugaredLogger, message pulsar.ProducerMessage)
}

type TableView interface {
	GetElement(log *zap.SugaredLogger)
}
