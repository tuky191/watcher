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
	GetBlock() (*block_feed.BlockResult, error)
}
