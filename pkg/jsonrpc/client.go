package jsonrpc

import (
	"net/http"
	"time"
		"encoding/json"
	"bytes"
	"io/ioutil"
	)

type Client struct {
	url    string
	client *http.Client
}

func NewClient(url string, timeout time.Duration) *Client {
	return &Client{
		url: url,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Execute(method string, params interface{}) (*Response, error) {
	if params == nil {
		params = []interface{}{}
	}

	serBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	id := time.Now().Unix()
	req := &Request{
		Jsonrpc: Version,
		Id:      id,
		Method:  method,
		Params:  serBody,
	}
	serReq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Post(c.url, "application/json", bytes.NewReader(serReq))
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
