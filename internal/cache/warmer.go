package cache

import (
	"github.com/inconshreveable/log15"
	"github.com/kyokan/chaind/pkg/log"
	"github.com/kyokan/chaind/internal/backend"
	"github.com/tidwall/gjson"
	"github.com/kyokan/chaind/pkg/concurrent"
	"sync/atomic"
	"strconv"
)

const EagerlyLoadedBlocks = 200
const WarmUpConcurrency = 5
const LastSeenKey = "lastseenblock"

type Warmer struct {
	store         *ETHStore
	cacher        Cacher
	hWatcher      *BlockHeightWatcher
	switcher      backend.Switcher
	hdl           int
	logger        log15.Logger
	lastSeenBlock uint64
}

func NewWarmer(store *ETHStore, cacher Cacher, hWatcher *BlockHeightWatcher, switcher backend.Switcher) *Warmer {
	return &Warmer{
		store:    store,
		cacher:   cacher,
		hWatcher: hWatcher,
		switcher: switcher,
		logger:   log.NewLog("warmer"),
	}
}

func (w *Warmer) Start() error {
	w.logger.Info("performing initial warmup")
	err := w.warm()
	if err != nil {
		return err
	}
	w.logger.Info("completed initial warmup")

	hdl := w.hWatcher.Subscribe(w.onBlock)
	w.hdl = hdl

	return nil
}

func (w *Warmer) Stop() error {
	w.hWatcher.Unsubscribe(w.hdl)
	return nil
}

func (w *Warmer) warm() error {
	client, err := w.switcher.ETHClient()
	if err != nil {
		return err
	}

	var lastSeenInCache uint64
	// can safely ignore errors here
	cached, _ := w.cacher.Get(LastSeenKey)
	if err != nil {
		w.logger.Warn("failed to get last seen key", "err", err)
	}
	if cached == nil {
		lastSeenInCache = 0
	} else {
		// can safely ignore errors here
		lastSeenInCache, _ = strconv.ParseUint(string(cached), 10, 64)
	}

	height, err := client.BlockNumber()
	if err != nil {
		return err
	}

	end := height - FinalityDepth

	var start uint64
	if lastSeenInCache > end-EagerlyLoadedBlocks {
		start = lastSeenInCache
	} else {
		start = end - EagerlyLoadedBlocks
	}

	if start > end {
		atomic.StoreUint64(&w.lastSeenBlock, start)
		w.logger.Info("cache already warm")
		return nil
	}

	w.cacheBlocksBetween(start, end)
	w.logger.Info("successfully warmed up cache", "start_block", start, "end_block", end)
	atomic.StoreUint64(&w.lastSeenBlock, end)
	if err := w.cacher.Set(LastSeenKey, []byte(strconv.FormatUint(end, 10))); err != nil {
		w.logger.Error("failed to store last seen block in cache", "err", err)
	}
	return nil
}

func (w *Warmer) onBlock(number uint64) {
	w.logger.Debug("got new block", "number", number)
	lastSeenBlock := atomic.LoadUint64(&w.lastSeenBlock)
	lastFinalized := number - FinalityDepth
	if lastFinalized < lastSeenBlock {
		w.logger.Debug("skipping non-finalized block", "number", number, "last_seen", lastSeenBlock)
		return
	}

	w.cacheBlocksBetween(lastSeenBlock, lastFinalized)
	atomic.StoreUint64(&w.lastSeenBlock, lastFinalized)
	if err := w.cacher.Set(LastSeenKey, []byte(strconv.FormatUint(number, 10))); err != nil {
		w.logger.Error("failed to store last seen block in cache", "err", err)
	}
}

func (w *Warmer) cacheBlocksBetween(start uint64, end uint64) {
	l := end - start
	if l == 0 {
		return
	}

	blocks := make([]uint64, l)
	for i := 0; i < int(l); i++ {
		blocks[i] = start + uint64(i)
	}
	concurrent.ConsumeUint64s(blocks, w.cacheBlock, WarmUpConcurrency)
}

func (w *Warmer) cacheBlock(number uint64) {
	client, err := w.switcher.ETHClient()
	if err != nil {
		w.logger.Error("failed to get Ethereum client", "err", err)
		return
	}

	blockRes, err := client.GetBlockByNumber(number, true)
	if err != nil {
		w.logger.Error("failed to get block by number", "err", err)
		return
	}

	if err := w.store.CacheBlockByNumber(blockRes, true); err != nil {
		w.logger.Error("failed to store block in cache", "err", err)
	}

	hashRes := gjson.GetBytes(blockRes, "transactions.#.hash").Array()
	l := len(hashRes)
	if l > 0 {
		txHashes := make([]string, l, l)
		for i, hash := range hashRes {
			txHashes[i] = hash.String()
		}
		go concurrent.ConsumeStrings(txHashes, w.cacheTxReceipt, WarmUpConcurrency)
	}

	w.logger.Debug("successfully warned up cache with block", "number", number)
}

func (w *Warmer) cacheTxReceipt(hash string) {
	client, err := w.switcher.ETHClient()
	if err != nil {
		w.logger.Error("failed to get Ethereum client", "err", err)
		return
	}

	receiptRes, err := client.GetTransactionReceipt(hash)
	if err != nil {
		w.logger.Error("failed to get transaction receipt", "err", err)
		return
	}

	if err := w.store.CacheTransactionReceipt(receiptRes); err != nil {
		w.logger.Error("failed to store tx receipt in cache", "err", err)
	}

	w.logger.Debug("successfully warmed up cache with transaction receipt", "tx_hash", hash)
}
