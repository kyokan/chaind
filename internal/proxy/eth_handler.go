package proxy

import (
	"net/http"
	"encoding/json"
	"github.com/kyokan/chaind/pkg"
	"github.com/kyokan/chaind/internal/cache"
	"github.com/inconshreveable/log15"
	"github.com/kyokan/chaind/pkg/log"
	"time"
	"io/ioutil"
	"bytes"
	"github.com/kyokan/chaind/internal/audit"
	"github.com/kyokan/chaind/pkg/jsonrpc"
	"github.com/kyokan/chaind/pkg/config"
	"strings"
	"github.com/kyokan/chaind/pkg/sets"
	"github.com/tidwall/gjson"
)

type beforeFunc func(res http.ResponseWriter, rpcReq *jsonrpc.Request, logger log15.Logger) bool
type afterFunc func(rpcRes *jsonrpc.Response, rpcReq *jsonrpc.Request, logger log15.Logger) error

type handler struct {
	before beforeFunc
	after  afterFunc
}

type EthHandler struct {
	store       *cache.ETHStore
	auditor     audit.Auditor
	hWatcher    *cache.BlockHeightWatcher
	handlers    map[string]*handler
	logger      log15.Logger
	client      *http.Client
	enabledAPIs *sets.StringSet
}

func NewEthHandler(store *cache.ETHStore, auditor audit.Auditor, hWatcher *cache.BlockHeightWatcher, enabledAPIs []string) *EthHandler {
	h := &EthHandler{
		store:    store,
		auditor:  auditor,
		hWatcher: hWatcher,
		logger:   log.NewLog("proxy/eth_handler"),
		client: &http.Client{
			Timeout: time.Second,
		},
		enabledAPIs: sets.NewStringSet(enabledAPIs),
	}
	h.handlers = map[string]*handler{
		"eth_blockNumber": {
			before: h.hdlBlockNumberBefore,
		},
		"eth_getBlockByNumber": {
			before: h.hdlGetBlockByNumberBefore,
			after:  h.hdlGetBlockByNumberAfter,
		},
		"eth_getTransactionReceipt": {
			before: h.hdlGetTransactionReceiptBefore,
			after:  h.hdlGetTransactionReceiptAfter,
		},
		"eth_getBalance": {
			before: h.hdlGetBalanceBefore,
			after:  h.hdlGetBalanceAfter,
		},
	}
	return h
}

func (h *EthHandler) Handle(res http.ResponseWriter, req *http.Request, backend *config.Backend) {
	defer req.Body.Close()
	logger := log.WithContext(h.logger, req.Context())
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error("failed to read request body")
		return
	}

	firstChar := string(body[0])
	// check if this is a batch request
	if firstChar == "[" {
		logger.Debug("got batch request")
		var rpcReqs []jsonrpc.Request
		err = json.Unmarshal(body, &rpcReqs)
		if err != nil {
			logger.Warn("received mal-formed batch request")
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		batch := pkg.NewBatchResponse(res)
		for _, rpcReq := range rpcReqs {
			h.hdlRPCRequest(batch.ResponseWriter(), req, backend, &rpcReq)
		}
		if err := batch.Flush(); err != nil {
			logger.Error("failed to flush batch")
		}

		logger.Debug("processed batch request")
	} else {
		logger.Debug("got single request")
		var rpcReq jsonrpc.Request
		err = json.Unmarshal(body, &rpcReq)
		if err != nil {
			logger.Warn("received mal-formed request")
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		h.hdlRPCRequest(res, req, backend, &rpcReq)
	}
}

func (h *EthHandler) hdlRPCRequest(res http.ResponseWriter, req *http.Request, backend *config.Backend, rpcReq *jsonrpc.Request) {
	logger := log.WithContext(h.logger, req.Context())
	body, err := json.Marshal(rpcReq)
	if err != nil {
		logger.Error("failed to unmarshal request body")
		return
	}

	err = h.auditor.RecordRequest(req, body, pkg.EthBackend)
	if err != nil {
		logger.Error("failed to record audit log for request")
	}

	split := strings.Split(rpcReq.Method, "_")
	if !h.enabledAPIs.Contains(split[0]) {
		failRequest(res, rpcReq.ID, -32602, "bad request")
		return
	}

	hdlr := h.handlers[rpcReq.Method]
	handledInBefore := false
	if hdlr != nil && hdlr.before != nil {
		handledInBefore = hdlr.before(res, rpcReq, logger)
	}
	if handledInBefore {
		logger.Debug("request handled in before filter")
		return
	}

	proxyRes, err := h.client.Post(backend.URL, "application/json", bytes.NewReader(body))
	if err != nil || proxyRes.StatusCode != 200 {
		failRequest(res, rpcReq.ID, -32602, "bad request")
		return
	}
	defer proxyRes.Body.Close()

	resBody, err := ioutil.ReadAll(proxyRes.Body)
	if err != nil {
		failWithInternalError(res, rpcReq.ID, err)
		logger.Error("failed to read body")
	}

	res.Write(resBody)
	if err != nil {
		logger.Error("failed to flush proxied request")
		failWithInternalError(res, rpcReq.ID, err)
		return
	}

	var rpcRes jsonrpc.Response
	err = json.Unmarshal(resBody, &rpcRes)
	if err != nil {
		logger.Debug("skipping post-processors for error response")
		return
	}

	if hdlr != nil && hdlr.after != nil {
		if err := hdlr.after(&rpcRes, rpcReq, logger); err != nil {
			logger.Error("request post-processing failed")
		}
	} else {
		logger.Debug("no post-processor found")
	}
}

func (h *EthHandler) hdlBlockNumberBefore(res http.ResponseWriter, rpcReq *jsonrpc.Request, logger log15.Logger) bool {
	logger.Debug("pre-processing eth_blockNumber")
	height := h.hWatcher.BlockHeight()
	if height == 0 {
		logger.Warn("received zero block height")
		return false
	}

	err := writeResponse(res, rpcReq.ID, []byte("\""+jsonrpc.Uint642Hex(height)+"\""))
	if err != nil {
		logger.Error("failed to write cached response")
		return false
	}
	logger.Debug("found cached block number response, sending")
	return true
}

func (h *EthHandler) hdlGetBlockByNumberBefore(res http.ResponseWriter, rpcReq *jsonrpc.Request, logger log15.Logger) bool {
	logger.Debug("pre-processing eth_getBlockByNumber")

	results := gjson.GetManyBytes(rpcReq.Params, "0", "1")
	blockNumStr := results[0].String()
	if blockNumStr == "" {
		logger.Info("encountered invalid block number param, bailing")
		return false
	}
	blockNum, err := jsonrpc.Hex2Uint64(blockNumStr)
	if err != nil {
		logger.Info("encountered invalid block number param, bailing")
		return false
	}

	includeBodies := results[1].Bool()
	cached, err := h.store.GetBlockByNumber(blockNum, includeBodies)
	if err == nil {
		if cached == nil {
			logger.Debug("found no blocks in block number cache")
			return false
		}

		err = writeResponse(res, rpcReq.ID, cached)
		if err != nil {
			logger.Error("failed to write cached response", "err", err)
			return false
		}

		logger.Debug("found cached block by number response, sending")
		return true
	}

	logger.Error("failed to get block from cache")
	return false
}

func (h *EthHandler) hdlGetBlockByNumberAfter(rpcRes *jsonrpc.Response, rpcReq *jsonrpc.Request, logger log15.Logger) error {
	logger.Debug("post-processing eth_getBlockByNumber")
	includeBodies := gjson.GetBytes(rpcReq.Params, "1").Bool()
	return h.store.CacheBlockByNumber(rpcRes.Result, includeBodies)
}

func (h *EthHandler) hdlGetTransactionReceiptBefore(res http.ResponseWriter, rpcReq *jsonrpc.Request, logger log15.Logger) bool {
	logger.Debug("pre-processing eth_getTransactionReceipt")

	txHash := gjson.GetBytes(rpcReq.Params, "0").String()
	if txHash == "" {
		logger.Debug("encountered invalid tx hash param, bailing")
		return false
	}

	cached, err := h.store.GetTransactionReceipt(txHash)
	if err == nil {
		if cached == nil {
			logger.Debug("found no tx receipts in tx receipt cache")
			return false
		}

		err = writeResponse(res, rpcReq.ID, cached)
		if err != nil {
			logger.Error("failed to write cached response", "err", err)
			return false
		}

		logger.Debug("found cached tx receipt response, sending")
		return true
	}

	logger.Error("failed to get tx receipt from cache")
	return false
}

func (h *EthHandler) hdlGetTransactionReceiptAfter(rpcRes *jsonrpc.Response, rpcReq *jsonrpc.Request, logger log15.Logger) error {
	logger.Debug("post-processing eth_getTransactionReceipt")
	return h.store.CacheTransactionReceipt(rpcRes.Result)
}

func (h *EthHandler) hdlGetBalanceBefore(res http.ResponseWriter, rpcReq *jsonrpc.Request, logger log15.Logger) bool {
	logger.Debug("pre-processing eth_getBalance")
	results := gjson.GetManyBytes(rpcReq.Params, "0", "1")
	if results[1].String() != "latest" {
		return false
	}

	addr := results[0].String()
	if addr == "" {
		logger.Info("encountered empty address, bailing")
		return false
	}
	cached, err := h.store.GetBalance(addr)
	if cached == nil {
		logger.Debug("no cached balance found")
		return false
	}
	err = writeResponse(res, rpcReq.ID, cached)
	if err != nil {
		logger.Error("encountered error writing response")
		return false
	}

	return true
}

func (h *EthHandler) hdlGetBalanceAfter(rpcRes *jsonrpc.Response, rpcReq *jsonrpc.Request, logger log15.Logger) error {
	h.logger.Debug("post-processing eth_getBalance")
	addr := gjson.GetBytes(rpcReq.Params, "0").String()
	if addr == "" {
		h.logger.Debug("skipping mal-formed address")
		return nil
	}

	return h.store.CacheBalance(addr, rpcRes.Result)
}

func writeResponse(res http.ResponseWriter, id interface{}, data []byte) error {
	outJson := &jsonrpc.Response{
		Jsonrpc: jsonrpc.Version,
		ID:      id,
		Result:  data,
	}

	out, err := json.Marshal(outJson)
	if err != nil {
		return err
	}
	res.Write(out)
	return nil
}

func failWithInternalError(res http.ResponseWriter, id interface{}, err error) {
	failRequest(res, id, -32600, err.Error())
}

func failRequest(res http.ResponseWriter, id interface{}, code int, msg string) {
	outJson := &jsonrpc.ErrorResponse{
		Version: jsonrpc.Version,
		ID:      id,
		Error: &jsonrpc.ErrorData{
			Code:    code,
			Message: msg,
		},
	}
	out, err := json.Marshal(outJson)
	if err != nil {
		out = []byte(jsonrpc.InternalError)
	}

	res.WriteHeader(http.StatusOK)
	res.Write(out)
}
