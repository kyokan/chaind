package jsonrpc

import (
	"testing"
	"github.com/stretchr/testify/require"
	"math/big"
)

func TestDe0x(t *testing.T) {
	require.Equal(t, "cabdab", De0x("0xcabdab"))
	require.Equal(t, "cabdab", De0x("cabdab"))
}

func TestHex2Big(t *testing.T) {
	val, err := Hex2Big("0x123")
	require.NoError(t, err)
	require.Equal(t, big.NewInt(291), val)

	_, err = Hex2Big("0xno")
	require.Error(t, err, "invalid hex string")
}

func TestHex2Uint64(t *testing.T) {
	val, err := Hex2Uint64("0x123")
	require.NoError(t, err)
	require.Equal(t, uint64(291), val)

	_, err = Hex2Uint64("0xno")
	require.Error(t, err, "invalid hex string")
}

func TestUint642Hex(t *testing.T) {
	hex := Uint642Hex(111)
	require.Equal(t, "0x6f", hex)
}