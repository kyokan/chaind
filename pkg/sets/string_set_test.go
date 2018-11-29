package sets

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestStringSet(t *testing.T) {
	set := NewStringSet([]string{
		"foo",
		"bar",
	})

	require.True(t, set.Contains("foo"))
	require.True(t, set.Contains("bar"))
	require.True(t, set.ContainsAll([]string{"foo", "bar"}))
	require.False(t, set.Contains("baz"))
	require.False(t, set.ContainsAll([]string{"foo", "baz"}))
}
