package jsonrpc

import (
	"encoding/json"
	)

const Version = "2.0"
const InternalError = "{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32603,\"message\":\"internal error\"}}"

type ErrorResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      interface{} `json:"id"`
	Error   *ErrorData  `json:"error"`
}

type ErrorData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Request struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result"`
}
