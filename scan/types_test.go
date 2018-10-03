package scan

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBitSet(t *testing.T) {
	var s bitSet
	require.Panics(t, func() { s.test(-1) })
	require.Panics(t, func() { s.test(bitSetSize) })
	for i := 0; i < bitSetSize; i++ {
		require.False(t, s.test(i), "i=%d", i)
		s.set(i)
	}
	for i := 0; i < bitSetSize; i++ {
		require.True(t, s.test(i), "i=%d", i)
	}
	for i := 0; i < bitSetSize; i++ {
		s = bitSet{}
		s.set(i)
		for j := 0; j < bitSetSize; j++ {
			require.Equal(t, i == j, s.test(j), "i=%d j=%d", i, j)
		}
	}
}
