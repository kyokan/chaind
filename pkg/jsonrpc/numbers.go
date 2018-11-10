package jsonrpc

import (
	"strings"
	"math/big"
	"github.com/pkg/errors"
	"fmt"
)


func De0x(num string) string {
	return strings.Replace(num, "0x", "", 1)
}

func Hex2Big(hex string) (*big.Int, error) {
	val, ok := new(big.Int).SetString(De0x(hex), 16)
	if !ok {
		return nil, errors.New("invalid hex string")
	}

	return val, nil
}

func Hex2Uint64(hex string) (uint64, error) {
	b, err := Hex2Big(hex)
	if err != nil {
		return 0, err
	}

	return b.Uint64(), err
}

func Uint642Hex(number uint64) string {
	return fmt.Sprintf("0x%s", new(big.Int).SetUint64(number).Text(16))
}