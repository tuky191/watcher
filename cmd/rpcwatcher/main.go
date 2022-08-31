package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/prestodb/presto-go-client/presto"
	"go.uber.org/zap"

	"rpc_watcher/rpcwatcher"

	_ "net/http/pprof"
	"rpc_watcher/rpcwatcher/database"
	"rpc_watcher/rpcwatcher/logging"
	"rpc_watcher/rpcwatcher/sync"

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
	db_config := &presto.Config{
		PrestoURI:         c.PrestoURI,
		SessionProperties: map[string]string{"catalog": "terra", "schema": chain.ChainName},
	}
	db, err := database.New(db_config)
	if err != nil {
		l.Errorw("Unable to create presto db handle", "error", err)
	}

	//spew.Dump(l)
	//sync_instance := sync.NewSync("de")

	sync_instance := sync.New(endpoint(chain.ChainName), l)
	_, err = sync.GetBlock(1, sync_instance)
	if err != nil {
		l.Errorw("Unable to get block", "url_string", "error", err)
	}
	watcher, cancel := startNewWatcher(chain.ChainName, c, l, db, false)
	watchers[chain.ChainName] = watcherInstance{
		watcher: watcher,
		cancel:  cancel,
	}

	for range time.Tick(1 * time.Second) {
		continue
	}

}

func startNewWatcher(chainName string, config *rpcwatcher.Config,
	l *zap.SugaredLogger, db *database.Instance, isNewChain bool) (*rpcwatcher.Watcher, context.CancelFunc) {
	eventMappings := rpcwatcher.StandardMappings

	/*
			for cn := range chainsMap {
			updatedChainsMap, watcher, cancel, shouldContinue := startNewWatcher(cn, chainsMap, c, s, l, false)
			chainsMap = updatedChainsMap
			if shouldContinue {
				continue
			}

			watchers[cn] = watcherInstance{
				watcher: watcher,
				cancel:  cancel,
			}
		}
	*/

	grpcEndpoint := fmt.Sprintf("%s:%d", "127.0.0.1", grpcPort)

	watcher, err := rpcwatcher.NewWatcher(config.RpcURL, chainName, l, config.ApiURL, grpcEndpoint, rpcwatcher.EventsToSubTo, eventMappings, config, db)
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
