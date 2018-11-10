package pkg

import (
	"net/http"
	"bytes"
)

type BatchResponse struct {
	writers     []*Interceptor
	res         http.ResponseWriter
}

func NewBatchResponse(res http.ResponseWriter) *BatchResponse {
	return &BatchResponse{
		res:         res,
	}
}

func (b *BatchResponse) ResponseWriter() http.ResponseWriter {
	interceptor := NewInterceptor()
	b.writers = append(b.writers, interceptor)
	return interceptor
}

func (b *BatchResponse) Flush() error {
	b.res.Write([]byte("["))
	for i, w := range b.writers {
		var buf bytes.Buffer
		n, err := w.buf.WriteTo(&buf)
		if err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		if i != 0 {
			b.res.Write([]byte(","))
		}
		buf.WriteTo(b.res)

	}
	b.res.Write([]byte("]"))
	return nil
}

type Interceptor struct {
	h          http.Header
	buf        bytes.Buffer
	statusCode int
}

func NewInterceptor() *Interceptor {
	var buf bytes.Buffer
	return &Interceptor{
		h:   make(http.Header),
		buf: buf,
	}
}

func (w *Interceptor) Header() http.Header {
	return w.h
}

func (w *Interceptor) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *Interceptor) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *Interceptor) Body() []byte {
	return w.buf.Bytes()
}

func (w *Interceptor) IsOK() bool {
	return w.statusCode == 200 || (w.statusCode == 0 && w.buf.Len() > 0)
}
