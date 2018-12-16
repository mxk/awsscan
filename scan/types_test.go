package scan

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpdateSkipFields(t *testing.T) {
	prev := skipFields
	defer func() { skipFields = prev }()
	skipFields = nil
	type output struct {
		_         struct{}
		A         int
		Truncated bool
		Input     string
		Output    string
		X         bool
		y         bool
		Z         bool
	}
	typ := reflect.TypeOf((*output)(nil)).Elem()
	updateSkipFields(typ, []string{"Input", "Output"}, "Truncated")
	want := typeBitSet{typ: {1<<0 | 1<<2 | 1<<3 | 1<<4 | 1<<6}}
	require.Equal(t, want, skipFields)
}

func TestBitSet(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	tests := []*struct {
		set  []int
		want bitSet
	}{
		{[]int{}, bitSet(nil)},
		{[]int{0}, bitSet{1}},
		{[]int{63}, bitSet{1 << 63}},
		{[]int{64}, bitSet{0, 1}},
		{[]int{126}, bitSet{0, 1 << 62}},
		{[]int{129}, bitSet{0, 0, 2}},
		{[]int{0, 63}, bitSet{1<<63 | 1}},
		{[]int{0, 65}, bitSet{1, 2}},
		{[]int{64, 1, 255}, bitSet{2, 1, 0, 1 << 63}},
		{[]int{1, 254, 256, 0, 65}, bitSet{3, 2, 0, 1 << 62, 1}},
		{rng.Perm(64), bitSet{math.MaxUint64}},
		{rng.Perm(128), bitSet{math.MaxUint64, math.MaxUint64}},
	}
	for _, tc := range tests {
		set := func(s *bitSet, i int) {
			require.False(t, s.test(i), "%+v i=%d", tc, i)
			s.set(i)
			require.True(t, s.test(i), "%+v i=%d", tc, i)
		}
		var fwd, rev, rnd bitSet
		perm := rng.Perm(len(tc.set))
		for i := range tc.set {
			set(&fwd, tc.set[i])
			set(&rev, tc.set[len(tc.set)-i-1])
			set(&rnd, tc.set[perm[i]])
		}
		require.Equal(t, tc.want, fwd, "%+v", tc)
		require.Equal(t, tc.want, rev, "%+v", tc)
		require.Equal(t, tc.want, rnd, "%+v", tc)
	}
	var s bitSet
	require.Panics(t, func() { s.test(-1) })
	require.Panics(t, func() { s.set(-1) })
}

func TestStaticBitSet(t *testing.T) {
	s := skipFields[nil]
	require.Zero(t, s)

	fs := staticBitSet{s[:]}
	fs.set(maxFields - 1)
	require.Panics(t, func() { fs.set(maxFields) })
	fs.set(maxFields - 64)

	require.Equal(t, make([]uint64, len(s)-1), s[:len(s)-1])
	require.Equal(t, uint64(1|1<<63), s[len(s)-1])
}
