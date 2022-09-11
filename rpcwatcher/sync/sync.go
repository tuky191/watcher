package sync

import (
	"rpc_watcher/rpcwatcher/database"

	watcher_pulsar "rpc_watcher/rpcwatcher/pulsar"

	block_feed "github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"
)

type SyncerOptions struct {
	Endpoint  string
	Logger    *zap.SugaredLogger
	Database  *database.Instance
	Producers map[string]watcher_pulsar.Producer
}

type Syncer interface {
	GetBlockByHeight(height int64) (*block_feed.BlockResult, error)
	GetLatestBlock() (*block_feed.BlockResult, error)
	GetBlock(query interface{}) (*block_feed.BlockResult, error)
	Run()
}
