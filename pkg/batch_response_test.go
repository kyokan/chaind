package pkg

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestInterceptor(t *testing.T) {
	body := []byte("[\"foo\"]")
	icept := NewInterceptor()
	icept.Header().Set("content-type", "application/json")
	icept.WriteHeader(200)
	icept.Write(body)

	require.Equal(t, "application/json", icept.Header().Get("content-type"))
	require.Equal(t, body, icept.Body())
	require.True(t, icept.IsOK())

	icept.WriteHeader(400)
	require.False(t, icept.IsOK())
}

func TestBatchResponse(t *testing.T) {
	icept := NewInterceptor()
	batch := NewBatchResponse(icept)
	bodies := [][]byte{
		[]byte("[\"foo\"]"),
		[]byte("[\"bar\"]"),
		{},
		[]byte("[\"baz\"]"),
		{},
	}
	for _, body := range bodies {
		batch.ResponseWriter().Write(body)
	}

	require.NoError(t, batch.Flush())
	require.Equal(t, "[[\"foo\"],[\"bar\"],[\"baz\"]]", string(icept.Body()))

	emptyIcept := NewInterceptor()
	emptyBatch := NewBatchResponse(emptyIcept)
	require.NoError(t, emptyBatch.Flush())
	require.Equal(t, "[]", string(emptyIcept.Body()))
}