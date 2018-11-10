package jsonrpc

import (
	"testing"
	"github.com/stretchr/testify/require"
)

const jsonDataObj = `
{
   "array":[
      1,
      2,
      3
   ],
   "boolean":true,
   "null":null,
   "number":123,
   "float":123.456,
   "object":{
      "a":"b",
      "c":"d",
      "e":"f",
      "g":[
         "honk",
         {
            "h":"i"
         }
      ]
   },
   "string":"Hello World",
   "bigHex": "0x1234"
}
`

const jsonDataArr = `
[
	{ 
		"key": "value"
	},
	2
]
`

func TestParsePatherObject(t *testing.T) {
	parsed, err := ParsePather([]byte(jsonDataObj))
	require.NoError(t, err)

	valI, err := parsed.GetInt("array.1")
	require.NoError(t, err)
	require.Equal(t, 2, valI)

	valI, err = parsed.GetInt("number")
	require.NoError(t, err)
	require.Equal(t, 123, valI)

	valStr, err := parsed.GetString("string")
	require.NoError(t, err)
	require.Equal(t, "Hello World", valStr)

	valStr, err = parsed.GetString("object.a")
	require.NoError(t, err)
	require.Equal(t, "b", valStr)

	valStr, err = parsed.GetString("object.g.1.h")
	require.NoError(t, err)
	require.Equal(t, "i", valStr)

	valBool, err := parsed.GetBool("boolean")
	require.NoError(t, err)
	require.True(t, valBool)

	valUintBig, err := parsed.GetHexUint("bigHex")
	require.NoError(t, err)
	require.Equal(t, uint64(4660), valUintBig)

	valLen, err := parsed.GetLen("array")
	require.NoError(t, err)
	require.Equal(t, 3, valLen)

	_, err = parsed.GetInt("null")
	require.Error(t, NullField, err)

	_, err = parsed.GetString("null")
	require.Error(t, NullField, err)

	_, err = parsed.GetBool("null")
	require.Error(t, NullField, err)

	parsedNil, err := ParsePather([]byte("null"))
	require.NoError(t, err)
	require.Nil(t, parsedNil)
}

func TestParsePathablerArray(t *testing.T) {
	parsed, err := ParsePather([]byte(jsonDataArr))
	require.NoError(t, err)

	valS, err := parsed.GetString("0.key")
	require.NoError(t, err)
	require.Equal(t, "value", valS)

	valI, err := parsed.GetInt("1")
	require.NoError(t, err)
	require.Equal(t, 2, valI)

	valLen, err := parsed.GetLen("")
	require.NoError(t, err)
	require.Equal(t, 2, valLen)
}
