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

func New(options SyncerOptions) *instance {

	ii := &instance{
		endpoint: options.Endpoint,
		logger:   options.Logger,
	}
	return ii
}

func (i *instance) GetBlock(height int64) (*block_feed.BlockResult, error) {
	ru, err := url.Parse(i.endpoint)
	if err != nil {
		i.logger.Errorw("cannot parse url", "url_string", i.endpoint, "error", err)
		return nil, err
	}
	vals := url.Values{}
	vals.Set("height", strconv.FormatInt(height, 10))

	i.logger.Debugw("asking for block", "height", height)

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
		i.logger.Errorw("endpoint returned non-200 code", "code", resp.StatusCode, "height", height)
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	return block_results, err
}
