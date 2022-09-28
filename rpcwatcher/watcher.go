package rpcwatcher

import (
	"context"
	"fmt"
	"rpc_watcher/rpcwatcher/avro"
	"rpc_watcher/rpcwatcher/database"
	syncer_types "rpc_watcher/rpcwatcher/helper"
	watcher_pulsar "rpc_watcher/rpcwatcher/pulsar"
	"strconv"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/apache/pulsar-client-go/pulsar"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/types"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	block_feed "github.com/terra-money/mantlemint/block_feed"

	"github.com/tendermint/tendermint/rpc/jsonrpc/client"
	"go.uber.org/zap"
)

const (
	EventsTx                = "tm.event='Tx'"
	EventsBlock             = "tm.event='NewBlock'"
	EventsTxTopic           = "tx"
	EventsBlockTopic        = "newblock"
	defaultWSClientReadWait = 30 * time.Second
	defaultWatchdogTimeout  = 20 * time.Second
	defaultReconnectionTime = 15 * time.Second
	defaultResubscribeSleep = 500 * time.Millisecond
	defaultTimeGap          = 750 * time.Millisecond
)

var (
	EventsToSubTo = []string{EventsTx, EventsBlock}
	EventTypeMap  = map[string]interface{}{
		EventsBlock: block_feed.BlockResult{},
		EventsTx:    abci.TxResult{},
	}
	TopicsMap = map[string]string{
		EventsBlock: EventsBlockTopic,
		EventsTx:    EventsTxTopic,
	}

	StandardMappings = map[string][]DataHandler{
		EventsTx: {
			HandleMessage,
		},
		EventsBlock: {
			HandleNewBlock,
		},
	}
)

type DataHandler func(watcher *Watcher, event coretypes.ResultEvent)

type Events map[string][]string

type Watcher struct {
	Name         string
	DataChannel  chan coretypes.ResultEvent
	ErrorChannel chan error

	eventTypeMappings map[string][]DataHandler
	apiUrl            string
	client            *client.WSClient
	l                 *zap.SugaredLogger
	db                *database.Instance
	sync              *syncer_types.Syncer
	Producers         map[string]watcher_pulsar.Producer
	runContext        context.Context
	endpoint          string
	grpcEndpoint      string
	subs              []string
	stopReadChannel   chan struct{}
	stopErrorChannel  chan struct{}
	watchdog          *watchdog
	config            *Config
}

func NewWatcher(
	endpoint, chainName string,
	logger *zap.SugaredLogger,
	apiUrl, grpcEndpoint string,
	subscriptions []string,
	eventTypeMappings map[string][]DataHandler,
	config *Config,
	database *database.Instance,
	sync syncer_types.Syncer,
) (*Watcher, error) {
	if len(eventTypeMappings) == 0 {
		return nil, fmt.Errorf("event type mappings cannot be empty")
	}
	producers := map[string]watcher_pulsar.Producer{}
	for _, eventKind := range subscriptions {

		schema, err := avro.GenerateAvroSchema(EventTypeMap[eventKind])
		properties := make(map[string]string)
		jsonSchemaWithProperties := pulsar.NewJSONSchema(schema, properties)
		o := watcher_pulsar.PulsarOptions{
			ClientOptions: pulsar.ClientOptions{
				URL:               config.PulsarURL,
				OperationTimeout:  30 * time.Second,
				ConnectionTimeout: 30 * time.Second,
			},
			ProducerOptions: pulsar.ProducerOptions{Topic: "persistent://terra/" + chainName + "/" + TopicsMap[eventKind], Schema: jsonSchemaWithProperties},
		}
		p, err := watcher_pulsar.NewProducer(&o)

		if err != nil {
			logger.Panicw("unable to start pulsar producer", "error", err)
		}
		producers[eventKind] = p

		handlers, ok := eventTypeMappings[eventKind]

		if !ok || len(handlers) == 0 {
			return nil, fmt.Errorf("event %s found in subscriptions but no handler defined for it", eventKind)
		}
	}
	ws, err := client.NewWS(
		endpoint,
		"/websocket",
		client.ReadWait(defaultWSClientReadWait),
	)

	if err != nil {
		return nil, err
	}

	ws.SetLogger(zapLogger{
		z:         logger,
		chainName: chainName,
	})

	if err := ws.OnStart(); err != nil {
		return nil, err
	}

	wd := newWatchdog(defaultWatchdogTimeout)

	w := &Watcher{
		apiUrl:            apiUrl,
		client:            ws,
		l:                 logger,
		db:                database,
		Producers:         producers,
		Name:              chainName,
		endpoint:          endpoint,
		grpcEndpoint:      grpcEndpoint,
		subs:              subscriptions,
		eventTypeMappings: eventTypeMappings,
		stopReadChannel:   make(chan struct{}),
		DataChannel:       make(chan coretypes.ResultEvent),
		stopErrorChannel:  make(chan struct{}),
		ErrorChannel:      make(chan error),
		watchdog:          wd,
		config:            config,
		sync:              &sync,
	}

	//w.producers[""].SendMessage()
	w.l.Debugw("creating rpcwatcher with config", "apiurl", apiUrl)
	sync.Run(true)

	for _, sub := range subscriptions {
		if err := w.client.Subscribe(context.Background(), sub); err != nil {
			return nil, fmt.Errorf("failed to subscribe, %w", err)
		}
	}

	wd.Start()

	go w.readChannel()
	go w.checkError()
	return w, nil
}

func Start(watcher *Watcher, ctx context.Context) {
	watcher.runContext = ctx
	go watcher.startChain(ctx)
}

func (w *Watcher) readChannel() {
	/*
		This thing uses nested selects because when we read from tendermint data channel, we should check first if
		the cancellation function has been called, and if yes we should return.

		Only after having done such check we can process the tendermint data.
	*/
	for {
		select {
		case <-w.stopReadChannel:
			return
		case <-w.watchdog.timeout:
			w.ErrorChannel <- fmt.Errorf("watchdog ticked, reconnect to websocket")
			return
		default:
			select {
			case data := <-w.client.ResponsesCh:
				if data.Error != nil {
					go func() {
						w.l.Debugw("writing error to error channel", "error", data.Error)
						w.ErrorChannel <- data.Error
					}()

					// if we get any kind of error from tendermint, exit: the reconnection routine will take care of
					// getting us up to speed again
					return
				}

				e := coretypes.ResultEvent{}
				if err := tmjson.Unmarshal(data.Result, &e); err != nil {
					w.l.Errorw("cannot unmarshal data into resultevent", "error", err, "chain", w.Name)
					continue
				}

				go func() {
					w.DataChannel <- e
				}()
			case <-time.After(defaultReconnectionTime):
				w.ErrorChannel <- fmt.Errorf("tendermint websocket hang, triggering reconnection")
				return
			}
		}
	}
}

func (w *Watcher) checkError() {
	for {
		select {
		case <-w.stopErrorChannel:
			return
		default:
			select { //nolint Intentional channel construct
			case err := <-w.ErrorChannel:
				if err != nil {
					w.l.Errorw("detected error", "chain_name", w.Name, "error", err)

				}
				resubscribe(w)
				return
			}
		}
	}
}

func resubscribe(w *Watcher) {
	count := 0
	for {
		w.l.Debugw("some error happened: resubscribing")

		time.Sleep(defaultResubscribeSleep)
		count++
		w.l.Debugw("this is count", "count", count)

		ww, err := NewWatcher(w.endpoint, w.Name, w.l, w.apiUrl, w.grpcEndpoint, w.subs, w.eventTypeMappings, w.config, w.db, *w.sync)
		if err != nil {
			w.l.Errorw("cannot resubscribe to chain", "name", w.Name, "endpoint", w.endpoint, "error", err)
			continue
		}
		ww.runContext = w.runContext
		w = ww
		Start(w, w.runContext)

		w.l.Infow("successfully reconnected", "name", w.Name, "endpoint", w.endpoint)
		return
	}
}

func (w *Watcher) startChain(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.stopReadChannel <- struct{}{}
			w.stopErrorChannel <- struct{}{}
			w.l.Infof("watcher %s has been canceled", w.Name)
			return
		default:
			select { //nolint intentional channel construct
			case data := <-w.DataChannel:
				if data.Query == "" {
					continue
				}

				handlers, ok := w.eventTypeMappings[data.Query]
				if !ok {
					w.l.Warnw("got event subscribed that didn't have a event mapping associated", "chain", w.Name, "eventName", data.Query)
					continue
				}
				for _, handler := range handlers {

					handler(w, data)
				}
			}
		}

	}
}

func HandleMessage(w *Watcher, data coretypes.ResultEvent) {
	txHashSlice := data.Events["tx.hash"]

	if len(txHashSlice) == 0 {
		return
	}

	txHash := txHashSlice[0]
	chainName := w.Name
	eventTx := data.Data.(types.EventDataTx)
	w.l.Debugw("Transaction Info", "chainName", chainName, "txHash", txHash, "log", eventTx.Result.Log)

	if eventTx.TxResult.Result.Events == nil {
		eventTx.TxResult.Result.Events = make([]abci.Event, 0)
	}

	message := pulsar.ProducerMessage{
		Value:       &eventTx.TxResult,
		SequenceID:  &eventTx.TxResult.Height,
		OrderingKey: strconv.FormatInt(eventTx.TxResult.Height, 10),
	}
	producer := w.Producers[data.Query]
	producer.SendMessage(w.l, message)

}

func HandleNewBlock(w *Watcher, data coretypes.ResultEvent) {
	w.watchdog.Ping()
	w.l.Debugw("performed watchdog ping", "chain_name", w.Name)
	w.l.Debugw("new block", "chain_name", w.Name)
	realData, ok := data.Data.(types.EventDataNewBlock)

	blockID := types.BlockID{
		Hash:          realData.Block.Hash(),
		PartSetHeader: realData.Block.MakePartSet(types.BlockPartSizeBytes).Header(),
	}
	BlockResults := block_feed.BlockResult{
		BlockID: &blockID,
		Block:   realData.Block,
	}

	if !ok {
		panic("rpc returned block data which is not of expected type")
	}

	if realData.Block == nil {
		w.l.Warnw("weird block received on rpc, it was empty while it shouldn't", "chain_name", w.Name)
	}
	if BlockResults.Block.Data.Txs == nil {
		BlockResults.Block.Data.Txs = make(types.Txs, 0)
	}
	if BlockResults.Block.Evidence.Evidence == nil {
		BlockResults.Block.Evidence.Evidence = make(types.EvidenceList, 0)
	}
	if BlockResults.Block.LastCommit.Signatures == nil {
		BlockResults.Block.LastCommit.Signatures = make([]types.CommitSig, 0)
	}
	message := pulsar.ProducerMessage{
		Value:       &BlockResults,
		SequenceID:  &realData.Block.Height,
		OrderingKey: strconv.FormatInt(realData.Block.Height, 10),
		EventTime:   BlockResults.Block.Time,
	}
	producer := w.Producers[data.Query]
	producer.SendMessage(w.l, message)
}
