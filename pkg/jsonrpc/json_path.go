package jsonrpc

import (
	"encoding/json"
	"strings"
	"github.com/pkg/errors"
	"strconv"
	"bytes"
	)

type JSONPather struct {
	data interface{}
}

var NullField = errors.New("null field")
var BadPath = errors.New("bad path")
var NullBody = []byte("null")

func ParsePather(message json.RawMessage) (*JSONPather, error) {
	if bytes.Equal(message, NullBody) {
		return &JSONPather{
			data: nil,
		}, nil
	}

	var data interface{}
	err := json.Unmarshal(message, &data)
	if err != nil {
		return nil, err
	}

	return &JSONPather{
		data: data,
	}, nil
}

func (p *JSONPather) GetInterface(path string) (interface{}, error) {
	if path == "" {
		return p.data, nil
	}

	parts := strings.Split(path, ".")

	if len(parts) == 0 {
		return nil, errors.New("no components in path")
	}

	var loc interface{}
	loc = p.data
	for i, part := range parts {
		idx, err := strconv.Atoi(part)
		isArray := err == nil

		if isArray {
			locArr, ok := loc.([]interface{})
			if !ok {
				return nil, errors.New("invalid path")
			}

			if idx > len(locArr)-1 {
				return nil, BadPath
			}

			if i == len(parts)-1 {
				return locArr[idx], nil
			}

			loc = locArr[idx]
		} else {
			locMap, ok := loc.(map[string]interface{})
			if !ok {
				return nil, BadPath
			}

			if i == len(parts)-1 {
				return locMap[part], nil
			}

			loc = locMap[part]
		}
	}

	return nil, errors.New("path does not result to final value")
}

func (p *JSONPather) GetBool(path string) (bool, error) {
	res, err := p.GetInterface(path)
	if err != nil {
		return false, err
	}
	out, ok := res.(bool)
	if !ok {
		if res == nil {
			return false, NullField
		}

		return false, errors.New("value is not a boolean")
	}
	return out, nil
}

func (p *JSONPather) GetString(path string) (string, error) {
	res, err := p.GetInterface(path)
	if err != nil {
		return "", err
	}
	out, ok := res.(string)
	if !ok {
		if res == nil {
			return "", NullField
		}

		return "", errors.New("value is not a string")
	}
	return out, nil
}

func (p *JSONPather) GetInt(path string) (int, error) {
	res, err := p.GetInterface(path)
	if err != nil {
		return 0, err
	}
	out, ok := res.(float64)
	if !ok {
		if res == nil {
			return 0, NullField
		}

		return 0, errors.New("value is not an integer")
	}
	return int(out), nil
}

func (p *JSONPather) GetHexUint(path string) (uint64, error) {
	str, err := p.GetString(path)
	if err != nil {
		return 0, err
	}

	return Hex2Uint64(str)
}

func (p *JSONPather) GetLen(path string) (int, error) {
	res, err := p.GetInterface(path)
	if err != nil {
		return 0, err
	}
	arr, ok := res.([]interface{})
	if !ok {
		if res == nil {
			return 0, NullField
		}

		return 0, errors.New("value is not an array")
	}

	return len(arr), nil
}

func (p *JSONPather) IsNil(path string) (bool, error) {
	res, err := p.GetInterface(path)
	if err != nil {
		return false, err
	}

	return res == nil, nil
}
