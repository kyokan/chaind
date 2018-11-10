package pkg

type BackendType string

const (
	EthBackend BackendType = "ETH"
	BtcBackend BackendType = "BTC"
)

type Backend struct {
	URL    string
	Name   string
	IsMain bool
	Type   BackendType
}
