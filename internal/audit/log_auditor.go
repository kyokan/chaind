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
	var rpcReq jsonrpc.Request
	err := json.Unmarshal(body, &rpcReq)
	if err != nil {
		l.logger.Error(
			"received request with invalid JSON body",
			mergeLogKeys(req, "type", pkg.EthBackend)...,
		)
		return nil
	}

	params, err := json.Marshal(rpcReq.Params)
	if err != nil {
		return err
	}
	l.logger.Info(
		"received JSON-RPC request",
		mergeLogKeys(req, "rpc_method", rpcReq.Method, "rpc_params", string(params))...,
	)
	return nil
}

func mergeLogKeys(req *http.Request, keys ... interface{}) []interface{} {
	defaults := []interface{}{
		"remote_addr",
		remoteAddr(req),
		"user_agent",
		req.Header.Get("user-agent"),
	}

	return log.WithRequestID(req.Context(), append(defaults, keys...)...)
}

func remoteAddr(req *http.Request) string {
	realIp := req.Header.Get("x-real-ip")
	if realIp != "" {
		return realIp
	}

	return req.RemoteAddr
}
