package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prestodb/presto-go-client/presto"
	"go.uber.org/zap"

	"rpc_watcher/rpcwatcher"
	rpc_types "rpc_watcher/rpcwatcher/helper/types/rpc"

	_ "net/http/pprof"
	"rpc_watcher/rpcwatcher/database"
	"rpc_watcher/rpcwatcher/helper/rpc"
	syncer_types "rpc_watcher/rpcwatcher/helper/types/syncer"
	"rpc_watcher/rpcwatcher/logging"
	"rpc_watcher/rpcwatcher/sync"
)

var Version = "0.01"

const grpcPort = 9090

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

	if err != nil {
		panic(err)
	}

	db_config := &presto.Config{
		PrestoURI: c.PrestoURI,
		Catalog:   "terra",
		Schema:    c.ChainID,
	}

	db, err := database.New(db_config)
	if err != nil {
		l.Errorw("Unable to create presto db handle", "error", err)
	}
	sync := sync.New(c, l)
	rpc := rpc.NewRPCApi(c, l)

	startNewWatcher(c, l, db, sync, rpc)

	for range time.Tick(1 * time.Second) {
		continue
	}

}

func startNewWatcher(config *rpcwatcher.Config,
	l *zap.SugaredLogger, db *database.Instance, sync syncer_types.Syncer, rpc rpc_types.Rpc) {
	eventMappings := rpcwatcher.StandardMappings

	grpcEndpoint := fmt.Sprintf("%s:%d", "127.0.0.1", grpcPort)

	watcher, err := rpcwatcher.NewWatcher(config.RpcURL, config.ChainID, l, config.ApiURL, grpcEndpoint, rpcwatcher.EventsToSubTo, eventMappings, config, db, sync, rpc)
	if err != nil {
		l.Errorw("cannot create chain", "error", err)
	}
	l.Debugw("connected", "chainName", config.ChainID)

	ctx := context.Background()
	rpcwatcher.Start(watcher, ctx)

}
