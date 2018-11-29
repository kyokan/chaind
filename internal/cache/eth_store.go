package cache

import (
	"github.com/kyokan/chaind/pkg/jsonrpc"
	"github.com/inconshreveable/log15"
	"errors"
	"time"
	"fmt"
	"github.com/tidwall/gjson"
	"strings"
	"strconv"
	"encoding/binary"
	"github.com/kyokan/chaind/pkg/log"
)

type ETHStore struct {
	cacher   Cacher
	hWatcher *BlockHeightWatcher
	logger   log15.Logger
}

func NewETHStore(cacher Cacher, hWatcher *BlockHeightWatcher) *ETHStore {
	return &ETHStore{
		cacher:   cacher,
		hWatcher: hWatcher,
		logger: log.NewLog("eth_store"),
	}
}

func (e *ETHStore) GetBlockByNumber(number uint64, includeBodies bool) ([]byte, error) {
	return e.cacher.Get(blockNumCacheKey(number, includeBodies))
}

func (e *ETHStore) CacheBlockByNumber(data []byte, includeBodies bool) error {
	results := gjson.GetManyBytes(data, "number", "transactions")
	if !results[0].Exists() {
		e.logger.Debug("skipping post-processing for null block")
		return nil
	}
	blockNumStr := results[0].String()
	blockNum, err := jsonrpc.Hex2Uint64(blockNumStr)
	if err != nil {
		e.logger.Error("encountered invalid block number, bailing")
		return nil
	}

	var expiry time.Duration
	if e.hWatcher.IsFinalized(blockNum) {
		expiry = time.Hour
	} else {
		e.logger.Debug("not caching un-finalized block", "number", blockNum)
		return nil
	}

	return e.cacher.SetEx(blockNumCacheKey(blockNum, includeBodies), data, expiry)
}

func (e *ETHStore) GetTransactionReceipt(hash string) ([]byte, error) {
	return e.cacher.Get(txReceiptCacheKey(hash))
}

func (e *ETHStore) CacheTransactionReceipt(data []byte) error {
	if !gjson.ParseBytes(data).Exists() {
		e.logger.Debug("skipping post-processing for null transaction")
		return nil
	}

	txHash := gjson.GetBytes(data, "transactionHash").String()
	blockNumStr := gjson.GetBytes(data, "blockNumber").String()
	if blockNumStr == "" {
		e.logger.Debug("skipping pending transaction", "tx_hash", txHash)
		return nil
	}
	blockNum, err := jsonrpc.Hex2Uint64(blockNumStr)
	if err != nil {
		return errors.New("failed to parse block number")
	}

	var expiry time.Duration
	if e.hWatcher.IsFinalized(blockNum) {
		expiry = time.Hour
	} else {
		e.logger.Debug("not caching un-finalized tx receipt", "hash", txHash, "number", blockNum)
		return nil
	}

	return e.cacher.SetEx(txReceiptCacheKey(txHash), data, expiry)
}

func (e *ETHStore) GetBalance(address string) ([]byte, error) {
	ck := balanceCacheKey(address)
	heightBytes, err := e.cacher.MapGet(ck, "blockNumber")
	if err != nil {
		return nil, err
	}
	height, _ := binary.Uvarint(heightBytes)
	if e.hWatcher.BlockHeight() > height {
		return nil, nil
	}

	cached, err := e.cacher.MapGet(ck, "balance")
	if err != nil {
		return nil, err
	}
	if cached == nil {
		return nil, errors.New("balance is nil, but blockNum isn't")
	}

	return cached, nil
}

func (e *ETHStore) CacheBalance(address string, data []byte) error {
	height := e.hWatcher.BlockHeight()
	var blockNumBytes [8]byte
	binary.PutUvarint(blockNumBytes[:], height)

	return e.cacher.MapSetEx(balanceCacheKey(address), map[string][]byte{
		"balance": data,
		"blockNumber": blockNumBytes[:],
	}, time.Minute)
}

func blockNumCacheKey(blockNum uint64, includeBodies bool) string {
	return fmt.Sprintf("block:%d:%s", blockNum, strconv.FormatBool(includeBodies))
}

func txReceiptCacheKey(hash string) string {
	return fmt.Sprintf("txreceipt:%s", strings.ToLower(hash))
}

func balanceCacheKey(addr string) string {
	return fmt.Sprintf("balance:%s:latest", strings.ToLower(addr))
}