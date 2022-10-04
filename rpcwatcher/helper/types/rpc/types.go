package rpc

import (
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/terra-money/mantlemint/block_feed"
	"github.com/tendermint/tendermint/types"

)

type Rpc interface {
	GetBlockByHeight(height int64) (*block_feed.BlockResult, error)
	GetLatestBlock() (*block_feed.BlockResult, error)
	GetBlock(query interface{}) (*block_feed.BlockResult, error)
	GetTxsFromBlockByHeight(height int64) []abci.TxResult
 	GetTx(txHashSlice types.Tx) abci.TxResult
}
