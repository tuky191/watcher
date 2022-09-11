package main

import (
	"rpc_watcher/rpcwatcher"

	"rpc_watcher/rpcwatcher/logging"
	"rpc_watcher/rpcwatcher/sync"
)

var Version = "0.01"

func main() {
	c, err := rpcwatcher.ReadConfig()
	if err != nil {
		panic(err)
	}
	l := logging.New(logging.LoggingConfig{
		Debug: c.Debug,
		JSON:  c.JSONLogs,
	})
	l.Infow("blocksync", "version", Version)

	sync := sync.New(c, l)
	sync.Run()

}
