package sync

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	block_feed "github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"
)

type instance struct {
	endpoint string
	logger   *zap.SugaredLogger
}
type block_query struct {
	height int64
	latest bool
}

func New(options SyncerOptions) *instance {

	ii := &instance{
		endpoint: options.Endpoint,
		logger:   options.Logger,
	}
	return ii
}

func (i *instance) GetBlockByHeight(height int64) (*block_feed.BlockResult, error) {
	result, err := i.GetBlock(block_query{
		height: height,
	})
	if err != nil {
		i.logger.Errorw("cannot get block from rcp", "block", height, "error", err)
		return nil, err
	}
	return result, nil
}

func (i *instance) GetLatestBlock() (*block_feed.BlockResult, error) {
	result, err := i.GetBlock(block_query{
		latest: true,
	})
	if err != nil {
		i.logger.Errorw("cannot get latest block", "error", err)
		return nil, err
	}
	return result, nil
}

func (i *instance) GetBlock(block_query block_query) (*block_feed.BlockResult, error) {

	ru, err := url.Parse(i.endpoint)
	if err != nil {
		i.logger.Errorw("cannot parse url", "url_string", i.endpoint, "error", err)
		return nil, err
	}
	vals := url.Values{}
	if !block_query.latest {
		vals.Set("height", strconv.FormatInt(block_query.height, 10))
		i.logger.Debugw("asking for block", "height", block_query.height)
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

	block_results, err := block_feed.ExtractBlockFromRPCResponse(body)

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
	return block_results, err
}
