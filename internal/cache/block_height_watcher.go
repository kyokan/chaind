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
	"github.com/kyokan/chaind/internal/health"
)

const FinalityDepth = 7

type BlockHeightWatcher struct {
	blockNumber uint64
	sw          health.BackendSwitch
	quitChan    chan bool
	logger      log15.Logger
	client      *http.Client
}

func NewBlockHeightWatcher(sw health.BackendSwitch) *BlockHeightWatcher {
	return &BlockHeightWatcher{
		sw:       sw,
		quitChan: make(chan bool),
		logger:   log.NewLog("proxy/block_number_watcher"),
		client: &http.Client{
			Timeout: time.Second,
		},
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
	return height-blockNum >= FinalityDepth
}

func (b *BlockHeightWatcher) BlockHeight() uint64 {
	return atomic.LoadUint64(&b.blockNumber)
}

func (b *BlockHeightWatcher) updateBlockHeight() {
	backend, err := b.sw.BackendFor(pkg.EthBackend)
	if err != nil {
		b.logger.Error("no backend available", "err", err)
	}

	client := jsonrpc.NewClient(backend.URL, time.Second)
	res, err := client.Execute("eth_blockNumber", nil)
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
	atomic.StoreUint64(&b.blockNumber, heightBig.Uint64())
}
