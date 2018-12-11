package cache

import (
	"time"
	"github.com/inconshreveable/log15"
	"github.com/kyokan/chaind/pkg/log"
	"github.com/kyokan/chaind/pkg"
	"net/http"
	"encoding/json"
	"sync/atomic"
	"github.com/kyokan/chaind/pkg/jsonrpc"
			"sync"
	"github.com/kyokan/chaind/internal/backend"
)

const FinalityDepth = 7

type BlockSub func(number uint64)

type BlockHeightWatcher struct {
	blockNumber uint64
	sw          backend.Switcher
	quitChan    chan bool
	logger      log15.Logger
	client      *http.Client
	subs        map[int]BlockSub
	lastSub     int
	subMu       sync.Mutex
}

func NewBlockHeightWatcher(sw backend.Switcher) *BlockHeightWatcher {
	return &BlockHeightWatcher{
		sw:       sw,
		quitChan: make(chan bool),
		logger:   log.NewLog("proxy/block_number_watcher"),
		client: pkg.NewHTTPClient(5 * time.Second),
		subs: make(map[int]BlockSub),
	}
}

func (b *BlockHeightWatcher) Start() error {
	b.updateBlockHeight()

	go func() {
		ticker := time.NewTicker(1 * time.Second)

		for {
			select {
			case <-ticker.C:
				b.updateBlockHeight()
			case <-b.quitChan:
				return
			}
		}
	}()

	return nil
}

func (b *BlockHeightWatcher) Stop() error {
	b.quitChan <- true
	return nil
}

func (b *BlockHeightWatcher) IsFinalized(blockNum uint64) bool {
	height := atomic.LoadUint64(&b.blockNumber)
	return height - FinalityDepth >= blockNum
}

func (b *BlockHeightWatcher) BlockHeight() uint64 {
	return atomic.LoadUint64(&b.blockNumber)
}

func (b *BlockHeightWatcher) Subscribe(cb BlockSub) int {
	b.subMu.Lock()
	defer b.subMu.Unlock()

	b.lastSub++
	b.subs[b.lastSub] = cb
	return b.lastSub
}

func (b *BlockHeightWatcher) Unsubscribe(hdl int) {
	b.subMu.Lock()
	defer b.subMu.Unlock()
	delete(b.subs, hdl)
}

func (b *BlockHeightWatcher) updateBlockHeight() {
	back, err := b.sw.BackendFor(pkg.EthBackend)
	if err != nil {
		b.logger.Error("no backend available", "err", err)
	}

	client := jsonrpc.NewClient(back.URL, time.Second)
	res, err := client.Call("eth_blockNumber")
	if err != nil {
		b.logger.Error("failed to fetch block height", "err", err)
		return
	}
	var heightStr string
	err = json.Unmarshal(res.Result, &heightStr)
	if err != nil {
		b.logger.Error("failed to unmarshal RPC result", "err", err)
		return
	}
	heightBig, err := jsonrpc.Hex2Big(heightStr)
	if err != nil {
		b.logger.Error("failed to create big num from response body", "err", err, "height", heightStr)
		return
	}

	b.logger.Debug("updated block height", "from", atomic.LoadUint64(&b.blockNumber), "to", heightBig.Uint64())
	height := heightBig.Uint64()
	atomic.StoreUint64(&b.blockNumber, height)
	go b.notifySubs(height)
}

func (b *BlockHeightWatcher) notifySubs(height uint64) {
	for _, sub := range b.subs {
		go sub(height)
	}
}