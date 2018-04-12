package service

import "reflect"

// setForEach implements the following loop:
//
//	for _, v := range src {
//		dst = append(dst, &newDstEntry{dstField: v.srcField})
//	}
//	return dst
//
// It does some additional work to make sure that the field assignment works as
// expected, such as appending v.srcField to a slice if that's the destination
// type. If srcField is empty, src is assumed to be a slice of values to assign
// directly.
func setForEach(dst interface{}, dstField string, src interface{}, srcField string) {
	sv := reflect.ValueOf(src)
	if sv.Len() == 0 {
		return
	}
	dt := reflect.TypeOf(dst).Elem().Elem().Elem()
	df, ok := dt.FieldByName(dstField)
	if !ok || len(df.Index) != 1 {
		panic("scan: dst field not found: " + dstField)
	}
	di := df.Index[0]

	const (
		passthrough = iota
		sliceDeref
		srcRef
	)
	method := passthrough
	if df.Type.Kind() == reflect.Slice && df.Type.Elem().Kind() != reflect.Ptr {
		method = sliceDeref
	}

	si := -1
	st := reflect.TypeOf(src).Elem()
	if srcField != "" {
		sf, ok := st.FieldByName(srcField)
		if !ok || len(sf.Index) != 1 {
			panic("scan: src field not found: " + srcField)
		}
		si = sf.Index[0]
	} else if st.Kind() != reflect.Ptr {
		method = srcRef
	}

	dv := reflect.ValueOf(dst).Elem()
	for i, n := 0, sv.Len(); i < n; i++ {
		ptr := reflect.New(dt)
		dst := ptr.Elem().Field(di)
		src := sv.Index(i)
		if si >= 0 {
			src = src.Field(si)
		}
		switch method {
		case passthrough:
		case sliceDeref:
			src = reflect.Append(dst, src.Elem())
		case srcRef:
			src = src.Addr()
		}
		dst.Set(src)
		dv = reflect.Append(dv, ptr)
	}
	reflect.ValueOf(dst).Elem().Set(dv)
}
