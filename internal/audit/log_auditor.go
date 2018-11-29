package audit

import (
	"github.com/inconshreveable/log15"
	"github.com/kyokan/chaind/pkg/config"
	"github.com/kyokan/chaind/pkg"
	"encoding/json"
	"github.com/pkg/errors"
	"net/http"
	"github.com/kyokan/chaind/pkg/jsonrpc"
	"github.com/kyokan/chaind/pkg/log"
)

type LogAuditor struct {
	logger log15.Logger
}

func NewLogAuditor(cfg *config.LogAuditorConfig) (Auditor, error) {
	if cfg == nil {
		return nil, errors.New("no log auditor config defined")
	}

	logger := log15.New()
	hdlr, err := log15.FileHandler(cfg.LogFile, log15.LogfmtFormat())
	if err != nil {
		return nil, err
	}
	logger.SetHandler(hdlr)

	return &LogAuditor{
		logger: logger,
	}, nil
}

func (l *LogAuditor) RecordRequest(req *http.Request, body []byte, reqType pkg.BackendType) error {
	if reqType == pkg.EthBackend {
		return l.recordETHRequest(req, body)
	}

	return nil
}

func (l *LogAuditor) recordETHRequest(req *http.Request, body []byte) error {
	logger := log.WithContext(l.logger.New("remote_addr", remoteAddr(req), "user_agent", req.Header.Get("user-agent")), req.Context())
	var rpcReq jsonrpc.Request
	err := json.Unmarshal(body, &rpcReq)
	if err != nil {
		logger.Error(
			"received request with invalid JSON body",
			"type",
			pkg.EthBackend,
		)
		return nil
	}

	params, err := json.Marshal(rpcReq.Params)
	if err != nil {
		return err
	}
	logger.Info(
		"received JSON-RPC request",
		"rpc_method", rpcReq.Method,
		"rpc_params", string(params),
	)
	return nil
}

func remoteAddr(req *http.Request) string {
	realIp := req.Header.Get("x-real-ip")
	if realIp != "" {
		return realIp
	}

	return req.RemoteAddr
}
