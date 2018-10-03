package scan

import (
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
	var s bitSet
	for i := t.NumField() - 1; i >= 0; i-- {
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

// typeBitSet associates a bitSet with a reflect.Type.
type typeBitSet map[reflect.Type]bitSet

// bitSet is an efficient map[int]bool representation.
type bitSet [1]uint64

const bitSetSize = len(bitSet{}) * 64

// test returns the status of bit i.
func (s *bitSet) test(i int) bool {
	idx, mask := idxMask(i)
	return s[idx]&mask != 0
}

// set sets bit i to true.
func (s *bitSet) set(i int) {
	idx, mask := idxMask(i)
	s[idx] |= mask
}

// idxMask returns bitSet array index and mask for bit i.
func idxMask(i int) (int, uint64) {
	return i >> 6, 1 << uint(i&63)
}
