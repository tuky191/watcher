package syncer_types

import (
	"rpc_watcher/rpcwatcher/database"
	pulsar_types "rpc_watcher/rpcwatcher/helper/types/pulsar"
	"time"

	block_feed "github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"
)

type SyncerOptions struct {
	Endpoint  string
	Logger    *zap.SugaredLogger
	Database  *database.Instance
	Producers map[string]pulsar_types.Producer
	Readers   map[string]pulsar_types.Reader
}

type Syncer interface {
	GetLatestPublishedBlockAndPublishTime() (*block_feed.BlockResult, time.Time, error)
	Run(sync_from_latest bool)
}
