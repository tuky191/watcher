package sync

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	tendermint "github.com/tendermint/tendermint/types"
	block_feed "github.com/terra-money/mantlemint/block_feed"
	"go.uber.org/zap"
)

type Instance struct {
	endpoint string
	logger   *zap.SugaredLogger
}

type BlockResult struct {
	BlockID *tendermint.BlockID `json:"block_id"`
	Block   *tendermint.Block   `json:"block"`
}

func New(endpoint string, l *zap.SugaredLogger) *Instance {

	ii := &Instance{
		endpoint: endpoint,
		logger:   l,
	}
	return ii
}

func GetBlock(height int64, s *Instance) (*block_feed.BlockResult, error) {
	ru, err := url.Parse(s.endpoint)
	if err != nil {
		s.logger.Errorw("cannot parse url", "url_string", s.endpoint, "error", err)
		return nil, err
	}
	vals := url.Values{}
	vals.Set("height", strconv.FormatInt(height, 10))

	s.logger.Debugw("asking for block", "height", height)

	ru.Path = "block"
	ru.RawQuery = vals.Encode()
	resp, err := http.Get(ru.String())
	if err != nil {
		s.logger.Errorw("failed to retrieve the response", "err", err)
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		s.logger.Errorw("failed to read the response", "err", err)
		return nil, err
	}

	block_results, err := block_feed.ExtractBlockFromRPCResponse(body)
	//spew.Dump(block_results)

	if err != nil {
		s.logger.Errorw("failed to unmarshal response", "err", err)

	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Errorw("endpoint returned non-200 code", "code", resp.StatusCode, "height", height)
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	return block_results, err
}
