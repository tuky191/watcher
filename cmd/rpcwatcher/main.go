package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"rpc_watcher/rpcwatcher"

	"rpc_watcher/rpcwatcher/logging"

	_ "net/http/pprof"
	producer "rpc_watcher/rpcwatcher/pulsar"
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

	cn := "localterra"
	watcher, cancel := startNewWatcher(cn, c, l)
	watchers := map[string]watcherInstance{}
	watchers[cn] = watcherInstance{
		watcher: watcher,
		cancel:  cancel,
	}
}

func startNewWatcher(chainName string, config *rpcwatcher.Config,
	l *zap.SugaredLogger) (*rpcwatcher.Watcher, context.CancelFunc) {
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
	//grpcEndpoint := fmt.Sprintf("%s:%d", chainName, grpcPort)
	grpcEndpoint := fmt.Sprintf("%s:%d", "127.0.0.1", grpcPort)
	if chainName == "localterra" { // special case, needs to observe new blocks too
		eventMappings = rpcwatcher.TerraMappings

		// caching node_info for localterra
		grpcConn, err := grpc.Dial(
			grpcEndpoint,
			grpc.WithInsecure(),
		)
		if err != nil {
			l.Errorw("cannot create gRPC client", "error", err, "chain name", chainName, "address", grpcEndpoint)
		}

		defer func() {
			if err := grpcConn.Close(); err != nil {
				l.Errorw("cannot close gRPC client", "error", err, "chain_name", chainName)
			}
		}()
		/*
			nodeInfoQuery := tmservice.NewServiceClient(grpcConn)
			nodeInfoRes, err := nodeInfoQuery.GetNodeInfo(context.Background(), &tmservice.GetNodeInfoRequest{})
			if err != nil {
				l.Errorw("cannot get node info", "error", err)
			}*/

	}

	watcher, err := rpcwatcher.NewWatcher(endpoint(chainName), chainName, l, config.ApiURL, grpcEndpoint, p, rpcwatcher.EventsToSubTo, eventMappings)
	if err != nil {
		l.Errorw("cannot create chain", "error", err)
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	rpcwatcher.Start(watcher, ctx)
	return watcher, cancel
}

func endpoint(chainName string) string {
	return "http://127.0.0.1:26657"
	//return fmt.Sprintf("http://%s:26657", chainName)
}
