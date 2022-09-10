package sync

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"rpc_watcher/rpcwatcher"
	"rpc_watcher/rpcwatcher/database"
	"strconv"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/prestodb/presto-go-client/presto"
	"github.com/tendermint/tendermint/types"
	block_feed "github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"
)

type instance struct {
	endpoint string
	logger   *zap.SugaredLogger
	db       *database.Instance
	w        *rpcwatcher.Watcher
}

func New(options SyncerOptions) Syncer {

	ii := &instance{
		endpoint: options.Endpoint,
		logger:   options.Logger,
		w:        options.Watcher,
	}
	return ii
}

func (i *instance) GetBlockByHeight(height int64) (*block_feed.BlockResult, error) {
	result, err := i.GetBlock(height)
	if err != nil {
		i.logger.Errorw("cannot get block from rcp", "block", height, "error", err)
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
	resp, err := http.Get(ru.String())
	if err != nil {
		i.logger.Errorw("failed to retrieve the response", "err", err)
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		i.logger.Errorw("failed to read the response", "err", err)
		return nil, err
	}

	block_result, err := block_feed.ExtractBlockFromRPCResponse(body)

	if err != nil {
		i.logger.Errorw("failed to unmarshal response", "err", err)

	}

	if resp.StatusCode != http.StatusOK {
		i.logger.Errorw("endpoint returned non-200 code", "code", resp.StatusCode)
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	//result := init_empty_slice(block_result)
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

// func init_empty_slice(s interface{}) interface{} {
// 	val := strctVal(s)
// 	if s.(*block_feed.BlockResult).Block.Data.Txs == nil {
// 		s.(*block_feed.BlockResult).Block.Data.Txs := make([]types.Txs, 0)
// 	}
// 	spew.Dump(s.(*block_feed.BlockResult).Block.Data.Txs)
// 	log.Fatal()
// 	v := val.Type()
// 	for i := 0; i < val.NumField(); i++ {
// 		t := v.Field(i)
// 		if val.Field(i).Kind() == reflect.Slice {
// 			spew.Dump(t)
// 		}

// 	}
// 	return s
// }

// func strctVal(s interface{}) reflect.Value {
// 	v := reflect.ValueOf(s)

// 	// if pointer get the underlying element
// 	for v.Kind() == reflect.Ptr {
// 		v = v.Elem()
// 	}

// 	return v
// }
func (i *instance) Init() {
	db_config := &presto.Config{
		PrestoURI: "http://root@localhost:8082",
	}

	db, err := database.New(db_config)
	if err != nil {
		i.logger.Errorw("Unable to create presto db handle", "error", err)
	}
	i.db = db

}

func (i *instance) Sync() {
	latest_block, err := i.GetLatestBlock()
	producer := i.w.Producers[rpcwatcher.EventsBlock]

	if err != nil {
		i.logger.Errorw("Unable to get latest block", "error", err)
	}
	current_height := latest_block.Block.Height
	min_height := 0
	published_blocks, err := i.db.Handle.Query(`select __sequence_id__ from pulsar."terra/localterra".newblock where __sequence_id__ > ` + fmt.Sprint(min_height) + ` and __sequence_id__ < ` + fmt.Sprint(current_height))
	if err != nil {
		i.logger.Errorw("Unable processed blocks from presto/pulsar", "error", err)
	}
	for published_blocks.Next() {
		var height int64
		if err := published_blocks.Scan(&height); err != nil {
			i.logger.Fatal(err)
		}
		fmt.Printf("height is %d\n", height)
		BlockResults, err := i.GetBlock(height)
		if err != nil {
			i.logger.Fatal(err)
		}
		message := pulsar.ProducerMessage{
			Value:       &BlockResults,
			SequenceID:  &BlockResults.Block.Height,
			OrderingKey: strconv.FormatInt(BlockResults.Block.Height, 10),
			EventTime:   BlockResults.Block.Time,
		}
		producer.SendMessage(i.logger, message)
	}
	if err := published_blocks.Err(); err != nil {
		log.Fatal(err)
	}

}
