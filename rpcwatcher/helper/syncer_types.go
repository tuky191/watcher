package syncer_types

import (
	"rpc_watcher/rpcwatcher/database"

	watcher_pulsar "rpc_watcher/rpcwatcher/pulsar"

	abci "github.com/tendermint/tendermint/abci/types"

	block_feed "github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"
)

type SyncerOptions struct {
	Endpoint  string
	Logger    *zap.SugaredLogger
	Database  *database.Instance
	Producers map[string]watcher_pulsar.Producer
	Readers   map[string]watcher_pulsar.Reader
}

type Syncer interface {
	GetBlockByHeight(height int64) (*block_feed.BlockResult, error)
	GetLatestBlock() (*block_feed.BlockResult, error)
	GetLatestPublishedBlock() (*block_feed.BlockResult, error)
	GetBlock(query interface{}) (*block_feed.BlockResult, error)
	GetTxsFromBlockByHeight(height int64) []abci.TxResult
	Run(sync_from_latest bool)
}
