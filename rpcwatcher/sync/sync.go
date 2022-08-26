package sync

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	tmjson "github.com/tendermint/tendermint/libs/json"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	typesjson "github.com/tendermint/tendermint/rpc/jsonrpc/types"
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

func GetBlock(height int64, s *Instance) (*ctypes.ResultEvent, error) {
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
	spew.Dump(block_results)

	rpc_response := new(typesjson.RPCResponse)
	err = tmjson.Unmarshal(body, rpc_response)

	if err != nil {
		s.logger.Errorw("failed to unmarshal response", "err", err)
		return nil, err
	}
	fmt.Printf("%s", rpc_response.Result)
	result := new(ctypes.ResultEvent)
	err = tmjson.Unmarshal(rpc_response.Result, result)
	if err != nil {
		s.logger.Errorw("failed to unmarshal response", "err", err)

	}
	//spew.Dump(result.Data)

	//spew.Dump(read)
	//realData, _ := result.Data.(types.EventDataNewBlock)
	//spew.Dump(realData.Block)
	//newHeight := realData.Block.Header.Height
	//spew.Dump(newHeight)

	if resp.StatusCode != http.StatusOK {
		s.logger.Errorw("endpoint returned non-200 code", "code", resp.StatusCode, "height", height)
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	return result, err
}
