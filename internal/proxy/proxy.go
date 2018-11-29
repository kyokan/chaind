package proxy

import (
	"github.com/kyokan/chaind/pkg"
	"github.com/kyokan/chaind/pkg/log"
	"github.com/kyokan/chaind/pkg/config"
	"net/http"
	"fmt"
	"context"
	"time"
	"github.com/kyokan/chaind/internal/audit"
	"github.com/satori/go.uuid"
	"github.com/kyokan/chaind/internal/cache"
	"github.com/kyokan/chaind/internal/backend"
)

var logger = log.NewLog("proxy")

type Proxy struct {
	sw         backend.Switcher
	config     *config.Config
	ethHandler *EthHandler
	quitChan   chan bool
	errChan    chan error
}

func NewProxy(sw backend.Switcher, auditor audit.Auditor, store *cache.ETHStore, fHelper *cache.BlockHeightWatcher, config *config.Config) *Proxy {
	return &Proxy{
		sw:         sw,
		config:     config,
		ethHandler: NewEthHandler(store, auditor, fHelper, config.ETHConfig.APIs),
		quitChan:   make(chan bool),
		errChan:    make(chan error),
	}
}

func (p *Proxy) Start() error {
	if p.config.UseTLS {
		panic("TLS not implemented yet")
	}

	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/%s", p.config.ETHConfig.Path), p.handleETHRequest)
	s := new(http.Server)
	s.Addr = fmt.Sprintf(":%d", p.config.RPCPort)
	s.Handler = mux

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("proxy server error", "port", p.config.RPCPort, "err", err)
		}
	}()

	go func() {
		<-p.quitChan
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.Shutdown(ctx); err != nil {
			p.errChan <- err
		}
		p.errChan <- nil
	}()

	logger.Info("started")
	return nil
}

func (p *Proxy) Stop() error {
	p.quitChan <- true
	return <-p.errChan
}

func (p *Proxy) handleETHRequest(res http.ResponseWriter, req *http.Request) {
	ctx := context.WithValue(req.Context(), log.RequestIDKey, uuid.NewV4().String())
	req = req.WithContext(ctx)
	cLog := log.WithContext(logger, req.Context())
	if req.Method != "POST" {
		cLog.Info("rejected non-POST request to eth endpoint")
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	back, err := p.sw.BackendFor(pkg.EthBackend)
	if err != nil {
		res.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	p.ethHandler.Handle(res, req, back)
	cLog.Info("finished handling Ethereum JSON-RPC request", "elapsed", time.Since(start))
}
