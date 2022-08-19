package rpcwatcher

import (
	"context"
	"fmt"
	producer "rpc_watcher/rpcwatcher/pulsar"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	tmjson "github.com/tendermint/tendermint/libs/json"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/rpc/jsonrpc/client"
	"github.com/tendermint/tendermint/types"
	"go.uber.org/zap"
)

const ackSuccess = "AQ==" // Packet ack value is true when ibc is success and contains error message in all other cases
const nonZeroCodeErrFmt = "non-zero code on chain %s: %s"

const (
	EventsTx                = "tm.event='Tx'"
	EventsBlock             = "tm.event='NewBlock'"
	defaultWSClientReadWait = 30 * time.Second
	defaultWatchdogTimeout  = 20 * time.Second
	defaultReconnectionTime = 15 * time.Second
	defaultResubscribeSleep = 500 * time.Millisecond
	defaultTimeGap          = 750 * time.Millisecond
)

var (
	EventsToSubTo = []string{EventsTx, EventsBlock}

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

type WsResponse struct {
	Event coretypes.ResultEvent `json:"result"`
}

type Events map[string][]string

type Watcher struct {
	Name         string
	DataChannel  chan coretypes.ResultEvent
	ErrorChannel chan error

	eventTypeMappings map[string][]DataHandler
	apiUrl            string
	client            *client.WSClient
	l                 *zap.SugaredLogger
	producer          map[string]producer.Instance
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
) (*Watcher, error) {
	if len(eventTypeMappings) == 0 {
		return nil, fmt.Errorf("event type mappings cannot be empty")
	}
	producers := map[string]producer.Instance{}
	for _, eventKind := range subscriptions {

		o := producer.Options{
			ClientOptions: pulsar.ClientOptions{
				URL:               config.PulsarURL,
				OperationTimeout:  30 * time.Second,
				ConnectionTimeout: 30 * time.Second,
			},
			ProducerOptions: pulsar.ProducerOptions{Topic: "persistent://terra/" + chainName + "/ws/" + eventKind},
		}
		p, err := producer.New(&o)

		if err != nil {
			logger.Panicw("unable to start pulsar producer", "error", err)
		}
		producers[eventKind] = *p
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
		producer:          producers,
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
	}

	w.l.Debugw("creating rpcwatcher with config", "apiurl", apiUrl)

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

		ww, err := NewWatcher(w.endpoint, w.Name, w.l, w.apiUrl, w.grpcEndpoint, w.subs, w.eventTypeMappings, w.config)
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
	txHashSlice, _ := data.Events["tx.hash"]

	if len(txHashSlice) == 0 {
		return
	}

	txHash := txHashSlice[0]
	chainName := w.Name
	eventTx := data.Data.(types.EventDataTx)
	w.l.Debugw("Transaction Info", "chainName", chainName, "txHash", txHash, "log", eventTx.Result.Log)
	producer.SendMessage(w.producer, w.l, data)
	//height := eventTx.Height

}

/*
	func HandleTerraBlock(w *Watcher, data coretypes.ResultEvent) {
		w.l.Debugw("called HandleTerraBlock")
		realData, ok := data.Data.(types.EventDataNewBlock)
		if !ok {
			panic("rpc returned data which is not of expected type")
		}

		time.Sleep(defaultTimeGap) // to handle the time gap between block production and event broadcast
		newHeight := realData.Block.Header.Height

		u := w.endpoint

		ru, err := url.Parse(u)
		if err != nil {
			w.l.Errorw("cannot parse url", "url_string", u, "error", err)
			return
		}

		vals := url.Values{}
		vals.Set("height", strconv.FormatInt(newHeight, 10))

		w.l.Debugw("asking for block", "height", newHeight)

		ru.Path = "block_results"
		ru.RawQuery = vals.Encode()

		res := bytes.Buffer{}
		spew.Dump(ru.String())

		resp, err := http.Get(ru.String())
		//spew.Dump(resp)
		if err != nil {
			w.l.Errorw("cannot query node for block data", "error", err, "height", newHeight)
			return
		}

		if resp.StatusCode != http.StatusOK {
			w.l.Errorw("endpoint returned non-200 code", "code", resp.StatusCode, "height", newHeight)
			return
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		read, err := res.ReadFrom(resp.Body)
		if err != nil {
			w.l.Errorw("cannot read block data resp body into buffer", "height", newHeight, "error", err)
			return
		}

		if read == 0 {
			w.l.Errorw("read zero bytes from response body", "height", newHeight)
		}

		// creating a grpc ClientConn to perform RPCs
		grpcConn, err := grpc.Dial(
			w.grpcEndpoint,
			grpc.WithInsecure(),
		)
		if err != nil {
			w.l.Errorw("cannot create gRPC client", "error", err, "chain_name", w.Name, "address", w.grpcEndpoint)
			return
		}

		defer func() {
			if err := grpcConn.Close(); err != nil {
				w.l.Errorw("cannot close gRPC client", "error", err, "chain_name", w.Name)
			}
		}()
		/*
			liquidityQuery := liquiditytypes.NewQueryClient(grpcConn)
			poolsRes, err := liquidityQuery.LiquidityPools(context.Background(), &liquiditytypes.QueryLiquidityPoolsRequest{})
			if err != nil {
				w.l.Errorw("cannot get liquidity pools in blocks", "error", err, "height", newHeight)
			}

			bz, err := w.store.Cdc.MarshalJSON(poolsRes)
			if err != nil {
				w.l.Errorw("cannot marshal liquidity pools", "error", err, "height", newHeight)
			}
			w.l.Infof(string(bz))

}
*/
func HandleNewBlock(w *Watcher, data coretypes.ResultEvent) {
	w.watchdog.Ping()
	w.l.Debugw("performed watchdog ping", "chain_name", w.Name)
	w.l.Debugw("new block", "chain_name", w.Name)

	realData, ok := data.Data.(types.EventDataNewBlock)
	//spew.Dump(realData)
	if !ok {
		panic("rpc returned block data which is not of expected type")
	}

	if realData.Block == nil {
		w.l.Warnw("weird block received on rpc, it was empty while it shouldn't", "chain_name", w.Name)
	}
	producer.SendMessage(w.producer, w.l, data)

}
