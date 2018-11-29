package backend

import (
	"github.com/kyokan/chaind/pkg/jsonrpc"
	"time"
	"github.com/tidwall/gjson"
	"github.com/pkg/errors"
	"encoding/json"
)

type ETHClient struct {
	client *jsonrpc.Client
}

func NewETHClient(url string) *ETHClient {
	return &ETHClient{
		client: jsonrpc.NewClient(url, 10 * time.Second),
	}
}

func (c *ETHClient) BlockNumber() (uint64, error) {
	res, err := c.client.Call("eth_blockNumber")
	if err != nil {
		return 0, err
	}

	blockNumberStr := gjson.ParseBytes(res.Result).String()
	if blockNumberStr == "" {
		return 0, errors.New("mal-formed block number")
	}

	return jsonrpc.Hex2Uint64(blockNumberStr)
}

func (c *ETHClient) GetTransactionReceipt(hash string) (json.RawMessage, error) {
	res, err := c.client.Call("eth_getTransactionReceipt", hash)
	if err != nil {
		return nil, err
	}

	return res.Result, nil
}

func (c *ETHClient) GetBlockByNumber(number uint64, includeBodies bool) (json.RawMessage, error) {
	res, err := c.client.Call("eth_getBlockByNumber", jsonrpc.Uint642Hex(number), includeBodies)
	if err != nil {
		return nil, err
	}

	return res.Result, nil
}