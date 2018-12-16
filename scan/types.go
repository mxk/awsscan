package scan

import (
	"math/bits"
	"reflect"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// skipFields contains bitSets for SDK Input/Output struct fields that identify
// which fields to skip when compacting a Map.
var (
	mu           sync.Mutex
	skipFields   typeBitSet
	nilPaginator aws.Paginator
)

// updateTypes extracts type information for Input/Output structs used by r.
func updateTypes(r *aws.Request) {
	it := reflect.TypeOf(r.Params).Elem()
	ot := reflect.TypeOf(r.Data).Elem()
	pg := r.Operation.Paginator
	if pg == nil {
		pg = &nilPaginator
	}
	mu.Lock()
	defer mu.Unlock()
	if _, ok := skipFields[it]; !ok {
		updateSkipFields(it, pg.InputTokens, pg.LimitToken)
	}
	if _, ok := skipFields[ot]; !ok {
		updateSkipFields(ot, pg.OutputTokens, pg.TruncationToken)
	}
}

// updateSkipFields adds a new bitSet to skipFields for type t identifying the
// fields to skip when compacting this type.
func updateSkipFields(t reflect.Type, ioToken []string, limTruncToken string) {
	s := skipFields[nil] // Zero value
	for i, s := t.NumField()-1, (staticBitSet{s[:]}); i >= 0; i-- {
		f := t.Field(i)
		if f.PkgPath != "" || f.Name == limTruncToken {
			s.set(i) // Unexported or Limit/Truncation token
			continue
		}
		for _, name := range ioToken {
			if f.Name == name {
				s.set(i) // Input/Output token
				break
			}
		}
	}
	if skipFields == nil {
		skipFields = make(typeBitSet)
	}
	skipFields[t] = s
}

// copySkipFields returns a copy of skipFields.
func copySkipFields() typeBitSet {
	mu.Lock()
	defer mu.Unlock()
	cpy := make(typeBitSet, len(skipFields))
	for t, s := range skipFields {
		cpy[t] = s
	}
	return cpy
}

// typeBitSet associates a reflect.Type with a fixed-size bitSet.
type typeBitSet map[reflect.Type][1]uint64

// maxFields is the maximum number of struct fields supported by typeBitSet.
const maxFields = len(typeBitSet(nil)[nil]) * 64

// bitSet is an efficient map[int]bool representation.
type bitSet []uint64

// test returns the status of bit i.
func (s bitSet) test(i int) bool {
	idx, mask := idxMask(i)
	return idx < len(s) && s[idx]&mask != 0
}

// set sets bit i to true.
func (s *bitSet) set(i int) {
	idx, mask := idxMask(i)
	if n := idx - len(*s); n >= 0 {
		*s = append(*s, make(bitSet, n+1)...)
	}
	(*s)[idx] |= mask
}

// len returns the number of bits set to true.
func (s bitSet) len() int {
	n := 0
	for _, x := range s {
		n += bits.OnesCount64(x)
	}
	return n
}

// staticBitSet prevents bitSet reallocation.
type staticBitSet struct{ bitSet }

// set sets bit i to true.
func (s staticBitSet) set(i int) {
	idx, mask := idxMask(i)
	s.bitSet[idx] |= mask
}

// idxMask returns bitSet slice index and mask for bit i.
func idxMask(i int) (int, uint64) {
	return i >> 6, 1 << uint(i&63)
}
