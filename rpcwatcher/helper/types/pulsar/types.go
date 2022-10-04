package pulsar_types

import (
	"github.com/apache/pulsar-client-go/pulsar"
	"go.uber.org/zap"
)

type PulsarOptions struct {
	ProducerOptions  pulsar.ProducerOptions
	ReaderOptions    pulsar.ReaderOptions
	ClientOptions    pulsar.ClientOptions
	TableViewOptions pulsar.TableViewOptions
}

type Producer interface {
	SendMessage(log *zap.SugaredLogger, message pulsar.ProducerMessage)
}
type Reader interface {
	ReadLastMessage(log *zap.SugaredLogger) (pulsar.Message, error)
}

type TableView interface {
	GetElement(log *zap.SugaredLogger)
}
