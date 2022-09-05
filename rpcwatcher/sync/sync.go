package sync

import (
	"rpc_watcher/rpcwatcher"

	block_feed "github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"
)

type SyncerOptions struct {
	Endpoint string
	Logger   *zap.SugaredLogger
	Watcher  *rpcwatcher.Watcher
}

type Syncer interface {
	GetBlockByHeight(height int64) (*block_feed.BlockResult, error)
	GetLatestBlock() (*block_feed.BlockResult, error)
	GetBlock(query interface{}) (*block_feed.BlockResult, error)
}
