package proxy

import (
	"github.com/stretchr/testify/mock"
	"github.com/kyokan/chaind/pkg"
	"github.com/kyokan/chaind/pkg/config"
	"net/http"
	"net/http/httptest"
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/require"
		"testing"
	)

type MockBackendSwitch struct {
	mock.Mock
	srv  *httptest.Server
	body []byte
}

func (m *MockBackendSwitch) SetBody(body []byte) {
	m.body = body
}

func (m *MockBackendSwitch) Start() error {
	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(m.body)
	}))
	return nil
}

func (m *MockBackendSwitch) Stop() error {
	m.srv.Close()
	return nil
}

func (m *MockBackendSwitch) BackendFor(t pkg.BackendType) (*config.Backend, error) {
	return &config.Backend{
		Type: pkg.EthBackend,
		URL:  m.srv.URL,
		Name: "test",
		Main: false,
	}, nil
}

type BlockHeightWatcherSuite struct {
	suite.Suite
	sw *MockBackendSwitch
	watcher *BlockHeightWatcher
}

func (s *BlockHeightWatcherSuite) SetupSuite() {
	s.sw = new(MockBackendSwitch)
	s.sw.SetBody([]byte("{\"jsonrpc\":\"2.0\",\"result\":\"0x123\",\"id\":1}"))
	require.NoError(s.T(), s.sw.Start())
	s.watcher = NewBlockHeightWatcher(s.sw)
	require.NoError(s.T(), s.watcher.Start())
}

func (s *BlockHeightWatcherSuite) TearDownSuite() {
	require.NoError(s.T(), s.sw.Stop())
	require.NoError(s.T(), s.watcher.Stop())
}

func (s *BlockHeightWatcherSuite) TestBlockHeight() {
	height := s.watcher.BlockHeight()
	require.Equal(s.T(), uint64(291), height)
}

func (s *BlockHeightWatcherSuite) TestIsFinalized() {
	require.True(s.T(), s.watcher.IsFinalized(284))
	require.False(s.T(), s.watcher.IsFinalized(285))
}

func TestBlockHeightWatcherSuite(t *testing.T) {
	suite.Run(t, new(BlockHeightWatcherSuite))
}