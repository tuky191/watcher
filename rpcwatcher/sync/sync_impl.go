package sync

import (
	"fmt"
	"rpc_watcher/rpcwatcher"
	"rpc_watcher/rpcwatcher/avro"
	"rpc_watcher/rpcwatcher/database"
	"rpc_watcher/rpcwatcher/helper/rpc"
	pulsar_types "rpc_watcher/rpcwatcher/helper/types/pulsar"
	rpc_types "rpc_watcher/rpcwatcher/helper/types/rpc"
	syncer_types "rpc_watcher/rpcwatcher/helper/types/syncer"
	watcher_pulsar "rpc_watcher/rpcwatcher/pulsar"

	"strconv"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/avast/retry-go"
	"github.com/prestodb/presto-go-client/presto"
	block_feed "github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type instance struct {
	endpoint   string
	is_syncing bool
	logger     *zap.SugaredLogger
	db         *database.Instance
	p          map[string]pulsar_types.Producer
	r          map[string]pulsar_types.Reader
	config     *rpcwatcher.Config
	rpc        rpc_types.Rpc
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

func InitBlocks(latest_published_height int64, height int64, size int64) []block_range {
	result := []block_range{}
	for i := int64(latest_published_height); i <= height; i += size {
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

func New(c *rpcwatcher.Config, l *zap.SugaredLogger) syncer_types.Syncer {
	db_config := &presto.Config{
		PrestoURI: c.PrestoURI,
		Catalog:   "terra",
		Schema:    c.ChainID,
	}

	presto_db, err := database.New(db_config)
	if err != nil {
		l.Errorw("Unable to create presto db handle", "error", err)
	}

	producers := map[string]pulsar_types.Producer{}
	readers := map[string]pulsar_types.Reader{}

	for _, eventKind := range rpcwatcher.EventsToSubTo {

		schema, err := avro.GenerateAvroSchema(rpcwatcher.EventTypeMap[eventKind])
		if err != nil {
			l.Panicw("unable to generate avro schema", "error", err, "event kind", eventKind)
		}
		properties := make(map[string]string)
		jsonSchemaWithProperties := pulsar.NewJSONSchema(schema, properties)
		o := pulsar_types.PulsarOptions{
			ClientOptions: pulsar.ClientOptions{
				URL:               c.PulsarURL,
				OperationTimeout:  30 * time.Second,
				ConnectionTimeout: 30 * time.Second,
			},
			ProducerOptions: pulsar.ProducerOptions{Topic: "persistent://terra/" + c.ChainID + "/" + rpcwatcher.TopicsMap[eventKind], Schema: jsonSchemaWithProperties},
			ReaderOptions:   pulsar.ReaderOptions{Topic: "persistent://terra/" + c.ChainID + "/" + rpcwatcher.TopicsMap[eventKind], Schema: jsonSchemaWithProperties, StartMessageID: pulsar.LatestMessageID(), StartMessageIDInclusive: true},
		}
		p, err := watcher_pulsar.NewProducer(&o)
		if err != nil {
			l.Panicw("unable to start pulsar producer", "error", err)
		}
		producers[eventKind] = p
		r, err := watcher_pulsar.NewReader(&o)
		if err != nil {
			l.Panicw("unable to create pulsar reader", "error", err)
		}
		readers[eventKind] = r

	}
	rpc := rpc.NewRPCApi(c, l)
	options := syncer_types.SyncerOptions{
		Endpoint:  c.RpcURL,
		Logger:    l,
		Database:  presto_db,
		Producers: producers,
		Readers:   readers,
	}
	ii := &instance{
		endpoint:   options.Endpoint,
		logger:     options.Logger,
		db:         options.Database,
		p:          options.Producers,
		r:          options.Readers,
		config:     c,
		rpc:        rpc,
		is_syncing: false,
	}
	return ii
}

func (i *instance) GetLatestPublishedBlockAndPublishTime() (*block_feed.BlockResult, time.Time, error) {
	result := &block_feed.BlockResult{}

	msg, err := i.r[rpcwatcher.EventsBlock].ReadLastMessage(i.logger)

	if err != nil {
		i.logger.Errorw("cannot get latest published block", "error", err)
		return nil, time.Now(), err
	}
	if msg == nil {
		return result, time.Now(), nil
	}
	msg.GetSchemaValue(&result)
	return result, msg.PublishTime(), nil
}

func (i *instance) getPublishedBlockHeights(min int64, max int64, publish_time time.Time) ([]int64, error) {
	blocks := []int64{}
	err := retry.Do(
		func() error {
			i.logger.Debugw(`select __sequence_id__ from pulsar."terra/` + i.config.ChainID + `".newblock where __sequence_id__ >= ` + fmt.Sprint(min) + ` and __sequence_id__ <= ` + fmt.Sprint(max) + ` and __publish_time__ >= timestamp ` + `'` + publish_time.Format("2006-01-02 03:04:05.000") + `'`)
			response, err := i.db.Handle.Query(`select __sequence_id__ from pulsar."terra/` + i.config.ChainID + `".newblock where __sequence_id__ >= ` + fmt.Sprint(min) + ` and __sequence_id__ <= ` + fmt.Sprint(max) + ` and __publish_time__ >= timestamp ` + `'` + publish_time.Format("2006-01-02 03:04:05.000") + `'`)
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
			}
			return nil
		},
		retry.OnRetry(func(n uint, err error) {
			i.logger.Warnw("Attempt:", "Retrying...", err)
		}),
		retry.Delay(time.Duration(10)*time.Second),
		retry.Attempts(10),
	)

	return blocks, err
}

func (i *instance) Run(sync_from_latest bool) {
	batch := int64(100000)

	i.is_syncing = true
	var latest_published_block_height int64
	var latest_block_height int64
	var publish_time time.Time
	if sync_from_latest {
		latest_published_block_height, publish_time = i.getLatestPublishedBlockHeightAndPublishTime()
	} else {
		latest_published_block_height = 0
	}

	latest_block_height = i.getLatestBlockHeight()

	for latest_published_block_height < latest_block_height {

		i.logger.Debugw("Latest block:", "block", latest_block_height)
		i.logger.Debugw("Latest published block:", "block", latest_published_block_height, "publish_time", publish_time)

		if latest_block_height < batch {
			batch = latest_block_height
		}

		blocks := InitBlocks(latest_published_block_height, latest_block_height, batch)

		for _, block_range := range blocks {
			published_blocks, err := i.getPublishedBlockHeights(block_range.min_height, block_range.max_height, publish_time)
			if err != nil {
				i.is_syncing = false
				i.logger.Fatalw("Failed to query pulsar sql for published blocks", "error", err)
			}
			for bl_index, block := range block_range.blocks {
				if slices.Contains(published_blocks, block.height) {
					i.logger.Debugw("Block is already published: ", "height", block.height)
					block_range.blocks[bl_index].published = true
				} else {
					i.logger.Debugw("Block has not been published yet: ", "height", block.height)
					BlockResults, err := i.rpc.GetBlockByHeight(block.height)
					if err != nil {
						i.is_syncing = false
						i.logger.Fatal(err)
					}
					if BlockResults != nil {
						message := pulsar.ProducerMessage{
							Value:       &BlockResults,
							SequenceID:  &BlockResults.Block.Height,
							OrderingKey: strconv.FormatInt(BlockResults.Block.Height, 10),
							EventTime:   BlockResults.Block.Time,
						}
						i.p[rpcwatcher.EventsBlock].SendMessage(i.logger, message)
						TxResults := i.rpc.GetTxsFromBlockByHeight(block.height)
						for _, txresult := range TxResults {
							message := pulsar.ProducerMessage{
								Value:       &txresult,
								SequenceID:  &txresult.Height,
								OrderingKey: strconv.FormatInt(txresult.Height, 10),
								EventTime:   BlockResults.Block.Time,
							}
							i.p[rpcwatcher.EventsTx].SendMessage(i.logger, message)
						}
						block_range.blocks[bl_index].published = true

					} else {
						i.logger.Error("Unable to get block %d", block.height)
					}

				}

			}
		}
		latest_published_block_height, publish_time = i.getLatestPublishedBlockHeightAndPublishTime()
		latest_block_height = i.getLatestBlockHeight()
	}

	i.is_syncing = false

}

func (i *instance) getLatestPublishedBlockHeightAndPublishTime() (int64, time.Time) {

	latest_published_block, publish_time, err := i.GetLatestPublishedBlockAndPublishTime()

	if err != nil {
		i.logger.Errorw("Unable to get latest published block", "error", err)
	}
	if latest_published_block.Block != nil {
		return latest_published_block.Block.Height, publish_time
	} else {
		return 0, time.Now()
	}
}

func (i *instance) getLatestBlockHeight() int64 {
	latest_block, err := i.rpc.GetLatestBlock()
	if err != nil {
		i.logger.Errorw("Unable to get latest block", "error", err)
	}
	return latest_block.Block.Height
}
