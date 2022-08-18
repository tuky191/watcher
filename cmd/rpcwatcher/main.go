package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"

	"rpc_watcher/rpcwatcher"
	"rpc_watcher/rpcwatcher/store"

	"rpc_watcher/rpcwatcher/logging"

	_ "net/http/pprof"
	producer "rpc_watcher/rpcwatcher/pulsar"

	cnsmodels "github.com/emerishq/demeris-backend-models/cns"
)

var Version = "0.01"

const grpcPort = 9090

type watcherInstance struct {
	watcher *rpcwatcher.Watcher
	cancel  context.CancelFunc
}

func main() {
	c, err := rpcwatcher.ReadConfig()
	if err != nil {
		panic(err)
	}

	l := logging.New(logging.LoggingConfig{
		Debug: c.Debug,
		JSON:  c.JSONLogs,
	})

	l.Infow("rpcwatcher", "version", Version)

	if c.Debug {
		go func() {
			l.Debugw("starting profiling server", "address", c.ProfilingServerURL)
			err := http.ListenAndServe(c.ProfilingServerURL, nil)
			if err != nil {
				l.Panicw("cannot run profiling server", "error", err)
			}
		}()
	}

	s, err := store.NewClient(c.RedisURL)
	if err != nil {
		l.Panicw("unable to start redis client", "error", err)
	}
	watchers := map[string]watcherInstance{}
	var chain = cnsmodels.Chain{
		ID:                  1,
		Enabled:             true,
		ChainName:           "localterra",
		Logo:                "localterra",
		DisplayName:         "localterra",
		PrimaryChannel:      map[string]string{},
		Denoms:              []cnsmodels.Denom{},
		DemerisAddresses:    []string{},
		GenesisHash:         "",
		NodeInfo:            cnsmodels.NodeInfo{},
		ValidBlockThresh:    0,
		DerivationPath:      "",
		SupportedWallets:    []string{},
		BlockExplorer:       "",
		PublicNodeEndpoints: cnsmodels.PublicNodeEndpoints{},
		CosmosSDKVersion:    Version,
	}

	if err != nil {
		spew.Dump(err)
		panic(err)
	}
	watcher, cancel := startNewWatcher(chain.ChainName, c, s, l, false)
	watchers[chain.ChainName] = watcherInstance{
		watcher: watcher,
		cancel:  cancel,
	}
	for range time.Tick(1 * time.Second) {
		continue
	}

}

func startNewWatcher(chainName string, config *rpcwatcher.Config, s *store.Store,
	l *zap.SugaredLogger, isNewChain bool) (*rpcwatcher.Watcher, context.CancelFunc) {
	eventMappings := rpcwatcher.StandardMappings
	client_options := producer.ClientOptions{
		URL:               "pulsar://localhost:6650",
		OperationTimeout:  30 * time.Second,
		ConnectionTimeout: 30 * time.Second,
	}
	producer_options := producer.ProducerOptions{Topic: chainName}
	p, err := producer.New(&client_options, &producer_options)

	if err != nil {
		l.Panicw("unable to start pulsar producer", "error", err)
	}
	grpcEndpoint := fmt.Sprintf("%s:%d", "127.0.0.1", grpcPort)

	watcher, err := rpcwatcher.NewWatcher(endpoint(chainName), chainName, l, config.ApiURL, grpcEndpoint, s, p, rpcwatcher.EventsToSubTo, eventMappings)
	if err != nil {
		l.Errorw("cannot create chain", "error", err)
		return nil, nil
	}
	l.Debugw("connected", "chainName", chainName)

	ctx, cancel := context.WithCancel(context.Background())
	rpcwatcher.Start(watcher, ctx)

	return watcher, cancel
}

func endpoint(chainName string) string {
	return "http://127.0.0.1:26657"
	//return fmt.Sprintf("http://%s:26657", chainName)
}
