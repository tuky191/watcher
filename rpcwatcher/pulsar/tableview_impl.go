package pulsar

import (
	"fmt"
	"log"
	pulsar_types "rpc_watcher/rpcwatcher/helper/types/pulsar"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

type TablewViewInstance struct {
	client     pulsar.Client
	TablewView pulsar.TableView
	o          pulsar_types.PulsarOptions
}

func NewTableView(o *pulsar_types.PulsarOptions) (pulsar_types.TableView, error) {
	client, err := pulsar.NewClient(o.ClientOptions)
	if err != nil {
		log.Fatalf("Could not instantiate Pulsar client: %v", err)
		return nil, err
	}

	table_view, err := NewTablewViewWithClient(client, o.TableViewOptions)
	if err != nil {
		log.Fatalf("Could not start the producer: %v", err)
		return nil, err
	}
	ii := &TablewViewInstance{
		client:     client,
		TablewView: table_view,
		o:          *o,
	}

	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	return ii, nil
}

func NewTablewViewWithClient(c pulsar.Client, o pulsar.TableViewOptions) (pulsar.TableView, error) {
	table_view, err := c.CreateTableView(o)
	if err != nil {
		log.Fatalf("Could not create table view: %v", err)
		return nil, err
	}
	return table_view, err
}

func (i *TablewViewInstance) GetElement(log *zap.SugaredLogger) {

	key := i.TablewView.Get("block")
	spew.Dump(key)
	for k, v := range i.TablewView.Entries() {
		spew.Dump(k)
		spew.Dump(v)
	}

	//spew.Dump(entries)
	//log.Debugw("submitted message in:", "data.Query", "with", "MessageId", fmt.Sprint(msg_id.EntryID()))

}
