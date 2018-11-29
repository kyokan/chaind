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
	"fmt"
	"strconv"
	"github.com/pkg/errors"
	"github.com/kyokan/chaind/internal/audit"
	"github.com/kyokan/chaind/pkg/jsonrpc"
	"github.com/kyokan/chaind/pkg/config"
	"encoding/binary"
	"strings"
	"github.com/kyokan/chaind/pkg/sets"
)

type beforeFunc func(res http.ResponseWriter, req *http.Request, rpcReq *jsonrpc.Request) bool
type afterFunc func(rpcRes *jsonrpc.Response, rpcReq *jsonrpc.Request, req *http.Request) error

type handler struct {
	before beforeFunc
	after  afterFunc
}

type EthHandler struct {
	cacher   cache.Cacher
	auditor  audit.Auditor
	hWatcher *BlockHeightWatcher
	handlers map[string]*handler
	logger   log15.Logger
	client   *http.Client
	enabledAPIs *sets.StringSet
}

func NewEthHandler(cacher cache.Cacher, auditor audit.Auditor, hWatcher *BlockHeightWatcher, enabledAPIs []string) *EthHandler {
	h := &EthHandler{
		cacher:   cacher,
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
			after: h.hdlGetBalanceAfter,
		},
	}
	return h
}

func (h *EthHandler) Handle(res http.ResponseWriter, req *http.Request, backend *config.Backend) {
	defer req.Body.Close()
	ctx := req.Context()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		h.logger.Error("failed to read request body", log.WithRequestID(ctx, "err", err)...)
		return
	}

	firstChar := string(body[0])
	// check if this is a batch request
	if firstChar == "[" {
		h.logger.Debug("got batch request", log.WithRequestID(ctx)...)
		var rpcReqs []jsonrpc.Request
		err = json.Unmarshal(body, &rpcReqs)
		if err != nil {
			h.logger.Warn("received mal-formed batch request", log.WithRequestID(ctx, "err", err)...)
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		batch := pkg.NewBatchResponse(res)
		for _, rpcReq := range rpcReqs {
			h.hdlRPCRequest(batch.ResponseWriter(), req, backend, &rpcReq)
		}
		if err := batch.Flush(); err != nil {
			h.logger.Error("failed to flush batch", log.WithRequestID(ctx, "err", err)...)
		}

		h.logger.Debug("processed batch request", log.WithRequestID(ctx, "count", len(rpcReqs))...)
	} else {
		h.logger.Debug("got single request", log.WithRequestID(ctx, "err", err)...)
		var rpcReq jsonrpc.Request
		err = json.Unmarshal(body, &rpcReq)
		if err != nil {
			h.logger.Warn("received mal-formed request", log.WithRequestID(ctx, "err", err)...)
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		h.hdlRPCRequest(res, req, backend, &rpcReq)
	}
}

func (h *EthHandler) hdlRPCRequest(res http.ResponseWriter, req *http.Request, backend *config.Backend, rpcReq *jsonrpc.Request) {
	ctx := req.Context()
	body, err := json.Marshal(rpcReq)
	if err != nil {
		h.logger.Error("failed to unmarshal request body", log.WithRequestID(ctx, "err", err)...)
		return
	}

	err = h.auditor.RecordRequest(req, body, pkg.EthBackend)
	if err != nil {
		h.logger.Error("failed to record audit log for request", log.WithRequestID(ctx, "err", err)...)
	}

	split := strings.Split(rpcReq.Method, "_")
	if !h.enabledAPIs.Contains(split[0]) {
		failRequest(res, rpcReq.Id, -32602, "bad request")
		return
	}

	hdlr := h.handlers[rpcReq.Method]
	handledInBefore := false
	if hdlr != nil && hdlr.before != nil {
		handledInBefore = hdlr.before(res, req, rpcReq)
	}
	if handledInBefore {
		h.logger.Debug("request handled in before filter", log.WithRequestID(ctx)...)
		return
	}

	proxyRes, err := h.client.Post(backend.URL, "application/json", bytes.NewReader(body))
	if err != nil || proxyRes.StatusCode != 200 {
		failRequest(res, rpcReq.Id, -32602, "bad request")
		return
	}
	defer proxyRes.Body.Close()

	resBody, err := ioutil.ReadAll(proxyRes.Body)
	if err != nil {
		failWithInternalError(res, rpcReq.Id, err)
		h.logger.Error("failed to read body", log.WithRequestID(ctx, "err", err))
	}

	res.Write(resBody)
	if err != nil {
		h.logger.Error("failed to flush proxied request", log.WithRequestID(ctx, "err", err)...)
		failWithInternalError(res, rpcReq.Id, err)
		return
	}

	var rpcRes jsonrpc.Response
	err = json.Unmarshal(resBody, &rpcRes)
	if err != nil {
		h.logger.Debug("skipping post-processors for error response", log.WithRequestID(ctx)...)
		return
	}

	if hdlr != nil && hdlr.after != nil {
		if err := hdlr.after(&rpcRes, rpcReq, req); err != nil {
			h.logger.Error("request post-processing failed", log.WithRequestID(ctx, "err", err)...)
		}
	} else {
		h.logger.Debug("no post-processor found", log.WithRequestID(ctx)...)
	}

}

func (h *EthHandler) hdlBlockNumberBefore(res http.ResponseWriter, req *http.Request, rpcReq *jsonrpc.Request) bool {
	ctx := req.Context()
	h.logger.Debug("pre-processing eth_blockNumber", log.WithRequestID(ctx)...)
	height := h.hWatcher.BlockHeight()
	if height == 0 {
		h.logger.Warn("received zero block height", log.WithRequestID(ctx)...)
		return false
	}

	err := writeResponse(res, rpcReq.Id, []byte("\""+jsonrpc.Uint642Hex(height)+"\""))
	if err != nil {
		h.logger.Error("failed to write cached response", log.WithRequestID(ctx, "err", err)...)
		return false
	}
	h.logger.Debug("found cached block number response, sending", log.WithRequestID(ctx)...)
	return true
}

func (h *EthHandler) hdlGetBlockByNumberBefore(res http.ResponseWriter, req *http.Request, rpcReq *jsonrpc.Request) bool {
	ctx := req.Context()
	h.logger.Debug("pre-processing eth_getBlockByNumber", log.WithRequestID(ctx)...)
	params := rpcReq.ParamsPather()
	paramCount, err := params.GetLen("")
	if err != nil {
		h.logger.Debug("encountered invalid params object", log.WithRequestID(ctx)...)
		return false
	}

	blockNum, err := params.GetHexUint("0")
	if err != nil {
		h.logger.Debug("encountered invalid block number param, bailing", log.WithRequestID(ctx, "block_num", blockNum)...)
		return false
	}

	var includeBodies bool
	if paramCount == 2 {
		testIncludeBodies, err := params.GetBool("1")
		if err != nil {
			h.logger.Debug("encountered invalid include bodies param, bailing", log.WithRequestID(ctx, "include_bodies", includeBodies)...)
			return false
		}
		includeBodies = testIncludeBodies
	}

	cacheKey := blockNumCacheKey(blockNum, includeBodies)
	h.logger.Debug("checking block number cache", log.WithRequestID(ctx, "cache_key", cacheKey)...)
	cached, err := h.cacher.Get(cacheKey)
	if err == nil && cached != nil {
		err = writeResponse(res, rpcReq.Id, cached)
		if err != nil {
			h.logger.Error("failed to write cached response", "err", err)
			return false
		}
		h.logger.Debug("found cached block by number response, sending", log.WithRequestID(ctx)...)
		return true
	}

	if err != nil {
		h.logger.Error("failed to get block from cache", log.WithRequestID(ctx, "err", err)...)
	}

	h.logger.Debug("found no blocks in block number cache", log.WithRequestID(ctx)...)
	return false
}

func (h *EthHandler) hdlGetBlockByNumberAfter(rpcRes *jsonrpc.Response, rpcReq *jsonrpc.Request, req *http.Request) error {
	ctx := req.Context()
	h.logger.Debug("post-processing eth_getBlockByNumber", log.WithRequestID(ctx)...)
	result := rpcRes.ResultPather()
	isNil, err := result.IsNil("")
	if err != nil {
		return err
	}
	if isNil {
		h.logger.Debug("skipping post-processing for null block")
		return nil
	}

	blockNum, err := result.GetHexUint("number")
	if err != nil {
		return errors.New("failed to parse block number from RPC results")
	}
	includeBodies, err := rpcReq.ParamsPather().GetBool("1")
	if err == jsonrpc.BadPath {
		includeBodies = false
	} else if err != nil {
		return err
	}

	var expiry time.Duration
	if h.hWatcher.IsFinalized(blockNum) {
		expiry = time.Hour
	} else {
		h.logger.Debug("not caching un-finalized block")
		return nil
	}

	cacheKey := blockNumCacheKey(blockNum, includeBodies)
	err = h.cacher.SetEx(cacheKey, rpcRes.Result, expiry)
	if err != nil {
		h.logger.Debug("post-processing failed while writing to cache", log.WithRequestID(ctx, "err", err)...)
		return err
	}
	h.logger.Debug("stored request in block number cache", log.WithRequestID(ctx, "cache_key", cacheKey, "size", len(rpcReq.Params))...)
	return nil
}

func (h *EthHandler) hdlGetTransactionReceiptBefore(res http.ResponseWriter, req *http.Request, rpcReq *jsonrpc.Request) bool {
	ctx := req.Context()
	h.logger.Debug("pre-processing eth_getTransactionReceipt", log.WithRequestID(ctx)...)
	params := rpcReq.ParamsPather()
	paramCount, err := params.GetLen("")
	if err != nil {
		h.logger.Error("encountered invalid request params", log.WithRequestID(ctx, "err", err)...)
		return false
	}
	if paramCount == 0 {
		return false
	}

	hash, err := params.GetString("0")
	if err != nil {
		h.logger.Debug("encountered invalid tx hash param, bailing", log.WithRequestID(ctx, "err", err)...)
		return false
	}

	cacheKey := txReceiptCacheKey(hash)
	h.logger.Debug("checking transaction receipt cache", log.WithRequestID(ctx, "cache_key", cacheKey)...)
	cached, err := h.cacher.Get(cacheKey)
	if err == nil && cached != nil {
		err = writeResponse(res, rpcReq.Id, cached)
		if err != nil {
			h.logger.Error("failed to write cached response", "err", err)
			return false
		}
		h.logger.Debug("found cached tx receipt response, sending", log.WithRequestID(ctx)...)
		return true
	}

	if err != nil {
		h.logger.Error("failed to get tx receipt from cache", log.WithRequestID(ctx, "err", err)...)
	}

	h.logger.Debug("found no tx receipts in tx receipt cache", log.WithRequestID(ctx)...)
	return false
}

func (h *EthHandler) hdlGetTransactionReceiptAfter(rpcRes *jsonrpc.Response, rpcReq *jsonrpc.Request, req *http.Request) error {
	ctx := req.Context()
	h.logger.Debug("post-processing eth_getTransactionReceipt", log.WithRequestID(ctx)...)
	result := rpcRes.ResultPather()
	isNil, err := result.IsNil("")
	if err != nil {
		return err
	}
	if isNil {
		h.logger.Debug("skipping post-processing for null transaction")
		return nil
	}

	txHash, err := result.GetString("transactionHash")
	if err != nil {
		return errors.New("failed to parse tx hash from RPC results")
	}
	blockNum, err := result.GetHexUint("blockNumber")
	if err != nil {
		if err == jsonrpc.NullField {
			h.logger.Debug("skipping pending transaction", log.WithRequestID(ctx)...)
			return nil
		}

		return errors.New("failed to parse block number from RPC results")
	}

	var expiry time.Duration
	if h.hWatcher.IsFinalized(blockNum) {
		expiry = time.Hour
	} else {
		h.logger.Debug("not caching un-finalized tx receipt")
		return nil
	}

	cacheKey := txReceiptCacheKey(txHash)
	err = h.cacher.SetEx(cacheKey, rpcRes.Result, expiry)
	if err != nil {
		h.logger.Debug("post-processing failed while writing to cache", log.WithRequestID(ctx, "err", err)...)
		return err
	}
	h.logger.Debug("stored request in tx receipt cache", log.WithRequestID(ctx, "cache_key", cacheKey, "size", len(rpcRes.Result))...)
	return nil
}

func (h *EthHandler) hdlGetBalanceBefore(res http.ResponseWriter, req *http.Request, rpcReq *jsonrpc.Request) bool {
	ctx := req.Context()
	h.logger.Debug("pre-processing eth_getBalance", log.WithRequestID(ctx)...)
	params := rpcReq.ParamsPather()
	reqBlockNum, err := params.GetString("1")
	if err != nil {
		h.logger.Debug("received invalid getBalance block number argument", log.WithRequestID(ctx, "err", err)...)
		return false
	}
	addr, err := params.GetString("0")
	if err != nil {
		h.logger.Debug("received invalid getBalance address argument", log.WithRequestID(ctx, "err", err)...)
		return false
	}

	if reqBlockNum != "latest" {
		return false
	}

	ck := balanceCacheKey(addr)
	cachedHeightBytes, err := h.cacher.MapGet(ck, "blockNumber")
	if err != nil {
		h.logger.Error("encountered error fetching blockNum from balance cache", log.WithRequestID(ctx, "err", err)...)
		return false
	}
	if cachedHeightBytes == nil {
		h.logger.Debug("no stored balance found for address", log.WithRequestID(ctx, "address", addr)...)
		return false
	}
	// should never overflow
	cachedHeight, _ := binary.Uvarint(cachedHeightBytes)
	if h.hWatcher.BlockHeight() > cachedHeight {
		return false
	}

	cached, err := h.cacher.MapGet(ck, "balance")
	if err != nil {
		h.logger.Error("encountered error fetching balance from balance cache", log.WithRequestID(ctx, "err", err)...)
		return false
	}
	if cached == nil {
		h.logger.Error("balance is nil, but blockNum isn't. should never happen", log.WithRequestID(ctx, "err", err)...)
		return false
	}

	err = writeResponse(res, rpcReq.Id, cached)
	if err != nil {
		h.logger.Error("encountered error writing response", log.WithRequestID(ctx, "err", err)...)
		return false
	}

	return true
}

func (h *EthHandler) hdlGetBalanceAfter(rpcRes *jsonrpc.Response, rpcReq *jsonrpc.Request, req *http.Request) error {
	ctx := req.Context()
	h.logger.Debug("post-processing eth_getBalance", log.WithRequestID(ctx)...)
	params := rpcReq.ParamsPather()
	addr, err := params.GetString("0")
	if err != nil {
		h.logger.Debug("skipping mal-formed address", log.WithRequestID(ctx, "err", err)...)
		return err
	}
	reqHeight, err := params.GetString("1")
	if err != nil {
		h.logger.Debug("skipping mal-formed request height", log.WithRequestID(ctx, "err", err)...)
		return err
	}
	if reqHeight != "latest" {
		h.logger.Debug("skipping non-latest block height balance", log.WithRequestID(ctx, "addr", addr, "req_height", reqHeight)...)
		return nil
	}

	height := h.hWatcher.BlockHeight()
	balance := rpcRes.Result
	var blockNumBytes [8]byte
	binary.PutUvarint(blockNumBytes[:], height)
	toCache := map[string][]byte {
		"balance": balance,
		"blockNumber": blockNumBytes[:],
	}
	cacheKey := balanceCacheKey(addr)
	h.logger.Debug("stored request in balance cache", log.WithRequestID(ctx, "cache_key", cacheKey, "size", len(rpcRes.Result))...)
	return h.cacher.MapSetEx(cacheKey, toCache, time.Minute)
}

func writeResponse(res http.ResponseWriter, id interface{}, data []byte) error {
	outJson := &jsonrpc.Response{
		Jsonrpc: jsonrpc.Version,
		Id:      id,
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
		Jsonrpc: jsonrpc.Version,
		Id:      id,
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

func blockNumCacheKey(blockNum uint64, includeBodies bool) string {
	return fmt.Sprintf("block:%d:%s", blockNum, strconv.FormatBool(includeBodies))
}

func txReceiptCacheKey(hash string) string {
	return fmt.Sprintf("txreceipt:%s", hash)
}

func balanceCacheKey(addr string) string {
	return fmt.Sprintf("balance:%s:latest", strings.ToLower(addr))
}