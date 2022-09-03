package sync

import (
	block_feed "github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"
)

type SyncerOptions struct {
	Endpoint string
	Logger   *zap.SugaredLogger
}

type Syncer interface {
	GetBlockByHeight(height int64) (*block_feed.BlockResult, error)
	GetLatestBlock() (*block_feed.BlockResult, error)
	GetBlock(block_query block_query) (*block_feed.BlockResult, error)
}
