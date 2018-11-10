package log

import (
	"github.com/inconshreveable/log15"
	"os"
	"context"
)

var rootLog = log15.New()

const DefaultLevel = log15.LvlInfo
const RequestIDKey = "request_id"

func init() {
	SetLevel(DefaultLevel)
}

func SetLevel(level log15.Lvl) {
	rootLog.SetHandler(log15.LvlFilterHandler(level, log15.StreamHandler(os.Stderr, log15.LogfmtFormat())))
}

func NewLog(module string) log15.Logger {
	if module == "" {
		return rootLog
	}

	return rootLog.New("module", module)
}

func WithRequestID(ctx context.Context, keys ... interface{}) []interface{} {
	return append(keys, []interface{}{
		"request_id",
		ctx.Value(RequestIDKey),
	}...)
}
