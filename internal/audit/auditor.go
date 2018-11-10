package audit

import (
	"github.com/kyokan/chaind/pkg"
	"net/http"
)

type Auditor interface {
	RecordRequest(req *http.Request, body []byte, reqType pkg.BackendType) error
}
