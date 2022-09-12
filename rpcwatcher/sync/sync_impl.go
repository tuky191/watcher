package sync

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"rpc_watcher/rpcwatcher"
	"rpc_watcher/rpcwatcher/avro"
	"rpc_watcher/rpcwatcher/database"
	watcher_pulsar "rpc_watcher/rpcwatcher/pulsar"
	"strconv"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/avast/retry-go"
	"github.com/prestodb/presto-go-client/presto"
	"github.com/tendermint/tendermint/types"
	block_feed "github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type instance struct {
	endpoint string
	logger   *zap.SugaredLogger
	db       *database.Instance
	p        map[string]watcher_pulsar.Producer
	config   *rpcwatcher.Config
}

type block struct {
	height    int64
	published bool
}
type block_range struct {
	min_height int64
	max_height int64
	blocks     []block
}

func InitBlocks(height int64, size int64) []block_range {
	result := []block_range{}
	for i := int64(0); i <= height; i += size {
		blocks := []block{}
		var limit int64
		if i+size < height {
			limit = i + size
		} else {
			limit = height
		}
		block_range := block_range{
			min_height: i + 1,
			max_height: limit,
		}
		for k := i + 1; k <= limit; k++ {
			blocks = append(blocks, block{height: k, published: false})
		}
		block_range.blocks = blocks
		result = append(result, block_range)
	}
	return result
}

func New(c *rpcwatcher.Config, l *zap.SugaredLogger) Syncer {
	db_config := &presto.Config{
		PrestoURI: c.PrestoURI,
		Catalog:   "terra",
		Schema:    c.ChainID,
	}

	presto_db, err := database.New(db_config)
	if err != nil {
		l.Errorw("Unable to create presto db handle", "error", err)
	}

	producers := map[string]watcher_pulsar.Producer{}
	for _, eventKind := range rpcwatcher.EventsToSubTo {

		schema, err := avro.GenerateAvroSchema(rpcwatcher.EventTypeMap[eventKind])
		if err != nil {
			l.Panicw("unable to generate avro schema", "error", err, "event kind", eventKind)
		}
		properties := make(map[string]string)
		jsonSchemaWithProperties := pulsar.NewJSONSchema(schema, properties)
		o := watcher_pulsar.PulsarOptions{
			ClientOptions: pulsar.ClientOptions{
				URL:               c.PulsarURL,
				OperationTimeout:  30 * time.Second,
				ConnectionTimeout: 30 * time.Second,
			},
			ProducerOptions: pulsar.ProducerOptions{Topic: "persistent://terra/" + c.ChainID + "/" + rpcwatcher.TopicsMap[eventKind], Schema: jsonSchemaWithProperties},
		}
		p, err := watcher_pulsar.NewProducer(&o)

		if err != nil {
			l.Panicw("unable to start pulsar producer", "error", err)
		}
		producers[eventKind] = p

	}
	options := SyncerOptions{
		Endpoint:  c.RpcURL,
		Logger:    l,
		Database:  presto_db,
		Producers: producers,
	}
	ii := &instance{
		endpoint: options.Endpoint,
		logger:   options.Logger,
		db:       options.Database,
		p:        options.Producers,
		config:   c,
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
	var body []byte
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
			fmt.Println("Attempt: ", n, ": Retrying...")
		}),
		retry.Delay(time.Duration(10)*time.Second),
		retry.Attempts(0),
	)

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

func (i *instance) getPublishedBlockHeights(min int64, max int64) ([]int64, error) {
	blocks := []int64{}
	err := retry.Do(
		func() error {
			response, err := i.db.Handle.Query(`select __sequence_id__ from pulsar."terra/` + i.config.ChainID + `".newblock where __sequence_id__ > ` + fmt.Sprint(min) + ` and __sequence_id__ < ` + fmt.Sprint(max))
			if err != nil {
				i.logger.Errorw("Unable to get processed blocks from presto/pulsar", "error", err)
				return err
			}
			for response.Next() {
				var height int64
				if err := response.Scan(&height); err != nil {
					i.logger.Fatal(err)
				}
				blocks = append(blocks, height)
				fmt.Printf("height is %d\n", height)
			}
			return nil
		},
		retry.OnRetry(func(n uint, err error) {
			fmt.Println("Attempt: ", n, ": Retrying...")
		}),
		retry.Delay(time.Duration(10)*time.Second),
		retry.Attempts(0),
	)

	return blocks, err
}

func (i *instance) Run() {
	batch := int64(300)
	producer := i.p[rpcwatcher.EventsBlock]

	latest_block, err := i.GetLatestBlock()
	if err != nil {
		i.logger.Errorw("Unable to get latest block", "error", err)
	}
	if latest_block.Block.Height < batch {
		batch = latest_block.Block.Height
	}
	blocks := InitBlocks(latest_block.Block.Height, batch)

	for _, block_range := range blocks {
		published_blocks, err := i.getPublishedBlockHeights(block_range.min_height, block_range.max_height)
		if err != nil {
			i.logger.Fatalw("Failed to query pulsar sql for published blocks", "error", err)
		}
		for bl_index, block := range block_range.blocks {
			if slices.Contains(published_blocks, block.height) {
				i.logger.Debugw("Block is already published: ", "height", block.height)
				block_range.blocks[bl_index].published = true
			} else {
				i.logger.Debugw("Block has not been published yet: ", "height", block.height)
				BlockResults, err := i.GetBlockByHeight(block.height)
				if err != nil {
					i.logger.Fatal(err)
				}
				if BlockResults != nil {
					message := pulsar.ProducerMessage{
						Value:       &BlockResults,
						SequenceID:  &BlockResults.Block.Height,
						OrderingKey: strconv.FormatInt(BlockResults.Block.Height, 10),
						EventTime:   BlockResults.Block.Time,
					}
					producer.SendMessage(i.logger, message)
					block_range.blocks[bl_index].published = true
				} else {
					i.logger.Error("Unable to get block %d", block.height)
				}
			}

		}
	}

}
