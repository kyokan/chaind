package backend

import (
	"net/http/httptest"
	"github.com/kyokan/chaind/pkg/config"
	"net/http"
	"github.com/kyokan/chaind/pkg"
	"testing"
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/require"
	"time"
	"sync"
	)

type BackendSwitchSuite struct {
	suite.Suite
	sw    Switcher
	srv1  *httptest.Server
	srv2  *httptest.Server
	code1 int
	body1 []byte
	code2 int
	body2 []byte
	mtx sync.Mutex
}

func (b *BackendSwitchSuite) SetupSuite() {
	b.srv1 = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.mtx.Lock()
		defer b.mtx.Unlock()
		if b.code1 != 0 {
			w.WriteHeader(b.code1)
		} else {
			w.Write(b.body1)
		}
	}))
	b.srv2 = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.mtx.Lock()
		defer b.mtx.Unlock()
		if b.code2 != 0 {
			w.WriteHeader(b.code2)
		} else {
			w.Write(b.body2)
		}
	}))

	b.body1 = []byte("{\"jsonrpc\":\"2.0\",\"result\":false,\"id\":1}")
	b.body2 = []byte("{\"jsonrpc\":\"2.0\",\"result\":false,\"id\":1}")

	b.sw = NewSwitcher([]config.Backend{
		{
			Name: "test-1",
			URL:  b.srv1.URL,
			Type: pkg.EthBackend,
		},
		{
			Name: "test-2",
			URL:  b.srv2.URL,
			Type: pkg.EthBackend,
			Main: true,
		},
	})

	require.NoError(b.T(), b.sw.Start())
}

func (b *BackendSwitchSuite) TearDownSuite() {
	b.srv1.Close()
	b.srv2.Close()
	require.NoError(b.T(), b.sw.Stop())
}

func (b *BackendSwitchSuite) TestBackendFor_A_InitialSuccess() {
	backend, err := b.sw.BackendFor(pkg.EthBackend)
	require.NoError(b.T(), err)
	require.Equal(b.T(), b.srv2.URL, backend.URL)
}

func (b *BackendSwitchSuite) TestBackendFor_B_AfterFailedHealthcheck() {
	b.mtx.Lock()
	b.code2 = http.StatusInternalServerError
	b.mtx.Unlock()
	time.Sleep(5000 * time.Millisecond)
	backend, err := b.sw.BackendFor(pkg.EthBackend)
	require.NoError(b.T(), err)
	require.Equal(b.T(), b.srv1.URL, backend.URL)
}

func (b *BackendSwitchSuite) TestBackendFor_C_NoMoreBackends() {
	b.mtx.Lock()
	b.code1 = http.StatusInternalServerError
	b.mtx.Unlock()
	time.Sleep(5000*time.Millisecond)
	backend, err := b.sw.BackendFor(pkg.EthBackend)
	require.Error(b.T(), err)
	require.Nil(b.T(), backend)
}

func TestBackendSwitchSuite(t *testing.T) {
	suite.Run(t, new(BackendSwitchSuite))
}
