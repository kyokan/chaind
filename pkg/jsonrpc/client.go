package jsonrpc

import (
	"net/http"
	"time"
	"encoding/json"
	"bytes"
	"io/ioutil"
	"sync/atomic"
	"github.com/kyokan/chaind/pkg"
)

type Client struct {
	url    string
	client *http.Client
	lastId int64
}

func NewClient(url string, timeout time.Duration) *Client {
	return &Client{
		url: url,
		client: pkg.NewHTTPClient(timeout),
	}
}

func (c *Client) Call(method string, params ... interface{}) (*Response, error) {
	if params == nil {
		params = []interface{}{}
	}

	serBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	id := atomic.AddInt64(&c.lastId, 1)
	req := &Request{
		Version: Version,
		ID:      id,
		Method:  method,
		Params:  serBody,
	}
	serReq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(serReq))
	httpReq.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	httpReq.Close = true
	res, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var rpcRes Response
	err = json.Unmarshal(resBody, &rpcRes)
	if err != nil {
		return nil, err
	}

	return &rpcRes, nil
}
