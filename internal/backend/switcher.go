package backend

import (
	"github.com/kyokan/chaind/pkg"
	"time"
	"github.com/inconshreveable/log15"
	"github.com/kyokan/chaind/pkg/log"
	"fmt"
	"net/http"
	"errors"
	"strings"
		"sync/atomic"
	"encoding/json"
	"github.com/kyokan/chaind/pkg/config"
	"sync"
		)

const ethCheckBody = "{\"jsonrpc\":\"2.0\",\"method\":\"eth_syncing\",\"params\":[],\"id\":%d}"

type Switcher interface {
	pkg.Service
	BackendFor(t pkg.BackendType) (*config.Backend, error)
	ETHClient() (*ETHClient, error)
}

type SwitcherImpl struct {
	ethBackends []config.Backend
	currEth     int32
	quitChan    chan bool
	logger      log15.Logger
}

func NewSwitcher(backendCfg []config.Backend) Switcher {
	var ethBackends []config.Backend
	var currEth int32

	for i, backend := range backendCfg {
		if backend.Type == pkg.EthBackend {
			ethBackends = append(ethBackends, backend)
		}

		if backend.Main {
			currEth = int32(i)
		}
	}

	return &SwitcherImpl{
		ethBackends: ethBackends,
		currEth:     currEth,
		quitChan:    make(chan bool),
		logger:      log.NewLog("proxy/backend_switch"),
	}
}

func (h *SwitcherImpl) Start() error {
	h.logger.Info("performing initial health checks on startup")
	h.performAllHealthchecks()

	go func() {
		tick := time.NewTicker(1 * time.Second)

		for {
			select {
			case <-tick.C:
				h.performAllHealthchecks()
			case <-h.quitChan:
				return
			}
		}
	}()

	return nil
}

func (h *SwitcherImpl) Stop() error {
	h.quitChan <- true
	return nil
}

func (h *SwitcherImpl) BackendFor(t pkg.BackendType) (*config.Backend, error) {
	var idx int32

	if t == pkg.EthBackend {
		idx = atomic.LoadInt32(&h.currEth)
	} else {
		return nil, errors.New("only Ethereum backends are supported")
	}

	if idx == -1 {
		return nil, errors.New("no backends available")
	}

	return &h.ethBackends[idx], nil
}

func (h *SwitcherImpl) ETHClient() (*ETHClient, error) {
	back, err := h.BackendFor(pkg.EthBackend)
	if err != nil {
		return nil, err
	}

	return NewETHClient(back.URL), nil
}

func (h *SwitcherImpl) performAllHealthchecks() {
	// use waitgroup so we can add btc checks later
	var wg sync.WaitGroup
	if h.currEth != -1 {
		wg.Add(1)
		go func() {
			idx := h.doHealthcheck(atomic.LoadInt32(&h.currEth), h.ethBackends)
			atomic.StoreInt32(&h.currEth, idx)
			wg.Done()
		}()
	}
	wg.Wait()
}

func (h *SwitcherImpl) doHealthcheck(idx int32, list []config.Backend) int32 {
	if idx == -1 {
		return -1
	}

	backend := list[idx]
	h.logger.Debug("performing healthcheck", "type", backend.Type, "name", backend.Name, "url", backend.URL)
	checker := NewChecker(&backend)
	ok := CheckWithBackoff(checker)

	if !ok {
		h.logger.Warn("backend is unhealthy, trying another", "type", backend.Type, "name", backend.Name, "url", backend.URL)
		return h.doHealthcheck(h.nextBackend(idx, list))
	}

	h.logger.Debug("backend is ok", "type", backend.Type, "name", backend.Name, "url", backend.URL)
	return idx
}

func (h *SwitcherImpl) nextBackend(idx int32, list []config.Backend) (int32, []config.Backend) {
	backend := list[idx]
	if len(list) == 1 || idx == int32(len(list) - 1) {
		h.logger.Error("no more backends to try", "type", backend.Type)
		return -1, list
	}

	if idx < int32(len(list)-1) {
		return idx + 1, list
	}

	return 0, list
}

func NewChecker(backend *config.Backend) Checker {
	if backend.Type == pkg.EthBackend {
		return &ETHChecker{
			backend: backend,
			logger:  log.NewLog("proxy/eth_checker"),
		}
	}

	return nil
}

type Checker interface {
	Check() bool
}

type ETHChecker struct {
	backend *config.Backend
	logger  log15.Logger
}

func (e *ETHChecker) Check() bool {
	id := time.Now().Unix()
	data := fmt.Sprintf(ethCheckBody, id)
	client := &http.Client{
		Timeout: time.Duration(2 * time.Second),
	}
	res, err := client.Post(e.backend.URL, "application/json", strings.NewReader(data))
	if err != nil {
		e.logger.Warn("backend returned non-200 response", "name", e.backend.Name, "url", e.backend.URL)
		return false
	}
	defer res.Body.Close()
	var dec map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&dec)
	if err != nil {
		e.logger.Warn("backend returned invalid JSON", "name", e.backend.Name, "url", e.backend.URL)
		return false
	}
	if _, ok := dec["result"].(bool); !ok {
		e.logger.Warn("backend is either completing initial sync or has fallen behind", "name", e.backend.Name, "url", e.backend.URL)
		return false
	}
	return true
}

func CheckWithBackoff(checker Checker) bool {
	count := 0

	for count < 3 {
		if checker.Check() {
			return true
		}
		count++
		time.Sleep(time.Second)
	}

	return false
}