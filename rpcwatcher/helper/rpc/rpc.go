package rpc

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"rpc_watcher/rpcwatcher"
	"strconv"
	"time"

	rpc_types "rpc_watcher/rpcwatcher/helper/types/rpc"

	"github.com/avast/retry-go"
	"github.com/tendermint/tendermint/types"
	"github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"

	abci "github.com/tendermint/tendermint/abci/types"

	tmjson "github.com/tendermint/tendermint/libs/json"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type instance struct {
	endpoint string
	logger   *zap.SugaredLogger
	config   *rpcwatcher.Config
}

func (i *instance) GetBlockByHeight(height int64) (*block_feed.BlockResult, error) {
	result, err := i.GetBlock(height)
	if err != nil {
		i.logger.Errorw("cannot get block from rpc", "block", height, "error", err)
		return nil, err
	}
	return result, nil
}
func (i *instance) GetLatestBlock() (*block_feed.BlockResult, error) {
	result, err := i.GetBlock(false)
	if err != nil {
		i.logger.Errorw("cannot get latest block", "error", err)
		return nil, err
	}
	return result, nil
}

func (i *instance) GetBlock(query interface{}) (*block_feed.BlockResult, error) {
	ru, err := url.Parse(i.endpoint)
	if err != nil {
		i.logger.Errorw("cannot parse url", "url_string", i.endpoint, "error", err)
		return nil, err
	}
	vals := url.Values{}
	if reflect.TypeOf(query).Kind() == reflect.Int64 {
		height := query.(int64)
		vals.Set("height", strconv.FormatInt(height, 10))
		i.logger.Debugw("asking for block", "height", height)
	}

	ru.Path = "block"
	ru.RawQuery = vals.Encode()

	body := i.fetchResponse(ru)
	block_result, err := block_feed.ExtractBlockFromRPCResponse(body)

	if err != nil {
		i.logger.Errorw("failed to unmarshal response", "err", err)

	}

	//Have to init nil slices to 0, to force json field change from "null" to "[]". Pulsar does accept avro schema ["null", "array"] union
	//for some reason so this is a workaround till i figure out how to handle this properly.
	if block_result.Block.Data.Txs == nil {
		block_result.Block.Data.Txs = make(types.Txs, 0)
	}
	if block_result.Block.Evidence.Evidence == nil {
		block_result.Block.Evidence.Evidence = make(types.EvidenceList, 0)
	}
	if block_result.Block.LastCommit.Signatures == nil {
		block_result.Block.LastCommit.Signatures = make([]types.CommitSig, 0)
	}

	return block_result, err
}

func (i *instance) fetchResponse(ru *url.URL) []byte {
	var body []byte
	retry.Do(
		func() error {
			resp, err := http.Get(ru.String())
			if err != nil {
				i.logger.Errorw("failed to retrieve the response", "err", err)
				return err
			}
			if resp.StatusCode != http.StatusOK {
				i.logger.Errorw("endpoint returned non-200 code", "code", resp.StatusCode)
				return err
			}

			defer func() {
				_ = resp.Body.Close()
			}()
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				i.logger.Errorw("failed to read the response", "err", err)
				return err
			}
			return nil
		},
		retry.OnRetry(func(n uint, err error) {
			i.logger.Warnw("Attempt:", "Retrying...", err)
		}),
		retry.Delay(time.Duration(10)*time.Second),
		retry.Attempts(3),
	)
	return body
}

func (i *instance) GetTxsFromBlockByHeight(height int64) []abci.TxResult {

	block, err := i.GetBlockByHeight(height)
	if err != nil {
		i.logger.Errorw("Unable to get block from rpc", "error", err)
	}

	Txs := make([]abci.TxResult, 0)
	for _, txHashSlice := range block.Block.Data.Txs {
		Txs = append(Txs, i.GetTx(txHashSlice))
	}
	return Txs
}

func (i *instance) GetTx(txHashSlice types.Tx) abci.TxResult {

	//curl -X GET "http://127.0.0.1:26657/tx?hash=0xABD1E4628F25750BB0E6DFEA827EC68730CEFDB118584AAF13DDC21440D473A2" -H  "accept: application/json"
	ru, err := url.Parse(i.endpoint)
	if err != nil {
		i.logger.Errorw("cannot parse url", "url_string", i.endpoint, "error", err)
	}
	vals := url.Values{}
	hash := fmt.Sprintf("0x%X", txHashSlice.Hash())

	vals.Set("hash", hash)
	i.logger.Debugw("asking for tx", "hash", hash)

	ru.Path = "tx"
	ru.RawQuery = vals.Encode()

	body := i.fetchResponse(ru)

	result_tx := new(struct {
		Result *coretypes.ResultTx `json:"result"`
	})
	if err := tmjson.Unmarshal(body, result_tx); err != nil {
		i.logger.Errorw("cannot extract tx result ", "error", err)
	}

	result := abci.TxResult{
		Height: result_tx.Result.Height,
		Index:  result_tx.Result.Index,
		Tx:     result_tx.Result.Tx,
		Result: result_tx.Result.TxResult,
	}
	return result
}
func NewRPCApi(c *rpcwatcher.Config, l *zap.SugaredLogger) rpc_types.Rpc {

	ii := &instance{
		endpoint: c.RpcURL,
		logger:   l,
		config:   c,
	}
	return ii
}
