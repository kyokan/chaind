package cache

import (
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/require"
	"github.com/satori/go.uuid"
	"time"
	)

type CacherSuite struct {
	suite.Suite
	cacher Cacher
}

func (c *CacherSuite) SetupSuite() {
	require.NoError(c.T(), c.cacher.Start())
}

func (c *CacherSuite) TearDownSuite() {
	require.NoError(c.T(), c.cacher.Stop())
}

func (c *CacherSuite) TestGetSetDel() {
	key := randStr()
	val := []byte(randStr())
	err := c.cacher.Set(key, val)
	require.NoError(c.T(), err)

	tVal, err := c.cacher.Get(key)
	require.NoError(c.T(), err)
	require.Equal(c.T(), val, tVal)

	nilVal, err := c.cacher.Get(randStr())
	require.NoError(c.T(), err)
	require.Nil(c.T(), nilVal)

	err = c.cacher.Del(key)
	require.NoError(c.T(), err)
	nilVal, err = c.cacher.Get(key)
	require.NoError(c.T(), err)
	require.Nil(c.T(), nilVal)
}

func (c *CacherSuite) TestSetEx() {
	key := randStr()
	val := []byte(randStr())
	err := c.cacher.SetEx(key, val, 10*time.Millisecond)
	require.NoError(c.T(), err)

	tVal, err := c.cacher.Get(key)
	require.NoError(c.T(), err)
	require.Equal(c.T(), val, tVal)

	time.Sleep(20 * time.Millisecond)
	nilVal, err := c.cacher.Get(key)
	require.NoError(c.T(), err)
	require.Nil(c.T(), nilVal)
}

func (c *CacherSuite) TestHas() {
	key := randStr()
	val := []byte(randStr())
	err := c.cacher.Set(key, val)
	require.NoError(c.T(), err)

	has, err := c.cacher.Has(key)
	require.NoError(c.T(), err)
	require.True(c.T(), has)

	err = c.cacher.Del(key)
	require.NoError(c.T(), err)
	has, err = c.cacher.Has(key)
	require.NoError(c.T(), err)
	require.False(c.T(), has)
}

func (c *CacherSuite) TestMapGetSetEx() {
	key := randStr()
	field1 := randStr()
	field2 := randStr()
	val1 := []byte(randStr())
	val2 := []byte(randStr())
	input := make(CacheableMap)
	input[field1] = val1
	input[field2] = val2
	err := c.cacher.MapSetEx(key, input, 10 * time.Millisecond)
	require.NoError(c.T(), err)

	tVal1, err := c.cacher.MapGet(key, field1)
	require.NoError(c.T(), err)
	require.Equal(c.T(), val1, tVal1)

	tVal2, err := c.cacher.MapGet(key, field2)
	require.NoError(c.T(), err)
	require.Equal(c.T(), val2, tVal2)

	nilVal, err := c.cacher.MapGet(key, randStr())
	require.NoError(c.T(), err)
	require.Nil(c.T(), nilVal)

	time.Sleep(20 * time.Millisecond)
	has, err := c.cacher.Has(key)
	require.NoError(c.T(), err)
	require.False(c.T(), has)
}

func randStr() string {
	return uuid.NewV4().String()
}
