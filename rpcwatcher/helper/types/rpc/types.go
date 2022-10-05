package rpc

import (
	"encoding/json"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/types"
	"github.com/terra-money/mantlemint/block_feed"
)

type TxRecord struct {
	Code      uint32          `json:"code"`
	Codespace string          `json:"codespace"`
	GasUsed   int64           `json:"gas_used"`
	GasWanted int64           `json:"gas_wanted"`
	Height    int64           `json:"height"`
	RawLog    string          `json:"raw_log"`
	Logs      json.RawMessage `json:"logs"`
	TxHash    string          `json:"txhash"`
	Timestamp time.Time       `json:"timestamp"`
	Tx        json.RawMessage `json:"tx"`
	Events    []abci.Event    `json:"events"`
}

type Rpc interface {
	GetBlockByHeight(height int64) (*block_feed.BlockResult, error)
	GetLatestBlock() (*block_feed.BlockResult, error)
	GetBlock(query interface{}) (*block_feed.BlockResult, error)
	GetTxsFromBlockByHeight(height int64) []abci.TxResult
	GetTx(txHashSlice types.Tx) abci.TxResult
}
