package scan

import (
	"bytes"
	"container/heap"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/LuminalHQ/cloudcover/x/arn"
	"github.com/LuminalHQ/cloudcover/x/tfx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	tf "github.com/hashicorp/terraform/terraform"
)

// Ctx contains the scan state for one service/region combination.
type Ctx struct {
	Map // Scan results

	mode   Mode             // Scan mode
	ars    string           // ID hash "<account>/<region>/<service>" prefix
	svc    *svc             // Service metadata
	iface  svcIface         // Service instance
	client reflect.Value    // SDK client instance
	run    map[*link]*batch // Run queue

	totalCalls int // Total number of calls made
	readyCalls int // Number of calls ready for execution
}

// newCtx creates a new scan context for the specified account/region/service.
func newCtx(cfg *aws.Config, ac arn.Ctx, svc *svc, opts Opts) *Ctx {
	cpy := cfg.Copy()
	cpy.Region = ac.Region
	ctx := &Ctx{
		Map: Map{
			Ctx:     ac,
			Service: svc.name,
			Calls:   make(map[string][]*Call, len(svc.api)),
		},
		mode:   opts.Mode,
		ars:    strings.Join([]string{ac.Account, ac.Region, svc.name}, "/"),
		svc:    svc,
		client: svc.newClient.Call([]reflect.Value{reflect.ValueOf(cpy)})[0],
		run:    make(map[*link]*batch, len(svc.links)),
	}
	iface := reflect.New(svc.typ).Elem()
	iface.FieldByName("Ctx").Set(reflect.ValueOf(ctx))
	ctx.iface = iface.Interface().(svcIface)
	return ctx
}

// TestCtx returns a context for unit testing a scanner implementation.
func TestCtx(iface interface{}) *Ctx {
	t := reflect.TypeOf(iface)
	for _, svc := range svcRegistry.get() {
		if svc.typ == t {
			cfg := defaults.Config()
			ac := arn.Ctx{"aws", "us-east-1", "123456789012"}
			return newCtx(&cfg, ac, svc, Opts{})
		}
	}
	panic("scan: " + t.String() + " is not registered")
}

// Mode returns true if all mode bits in test are enabled.
func (ctx *Ctx) Mode(test Mode) bool { return ctx.mode&test == test }

// SetMode sets scan mode. This should only be used for testing.
func (ctx *Ctx) SetMode(m Mode) { ctx.mode = m }

// UpdateRequest can be overridden by service scanners to modify each request
// before it is sent to fix broken SDK models.
func (*Ctx) UpdateRequest(*aws.Request) {}

// HandleError can be overridden by service scanners to set err.Ignore flag if
// the error is part of normal service behavior and is safe to ignore.
func (*Ctx) HandleError(*aws.Request, *Err) {}

// ARN constructs a service-specific ARN for the specified resource.
func (ctx *Ctx) ARN(resource ...string) *string {
	return arn.String(ctx.New(ctx.svc.id, resource...))
}

// Split populates one non-slice Input struct field in *dst for each src slice
// value. If srcField is specified, src must be a slice of structs or struct
// pointers containing the named field.
func (*Ctx) Split(dst interface{}, dstField string, src interface{}, srcField string) {
	sv, n := sliceValue(src)
	if n == 0 {
		return
	}
	dv := ensureSliceAt(dst, n)
	df, _ := fieldByName(dv.Type(), dstField)
	sf := -1
	if srcField != "" {
		sf, _ = fieldByName(sv.Type(), srcField)
	}
	for i := 0; i < n; i++ {
		dv.Index(i).Field(df).Set(ptrTo(sv, i, sf))
	}
}

// Group populates one slice Input struct field in *dst using up to max src
// slice values. If max is <= 0, then all src values are added to one Input
// struct. If srcField is specified, src must be a slice of structs or struct
// pointers containing the named field.
func (*Ctx) Group(dst interface{}, dstField string, src interface{}, srcField string, max int) {
	sv, n := sliceValue(src)
	if n == 0 {
		return
	}
	if max <= 0 {
		max = n
	}
	g := (n + (max - 1)) / max
	dv := ensureSliceAt(dst, g)
	df, dt := fieldByName(dv.Type(), dstField)
	sf := -1
	if srcField != "" {
		sf, _ = fieldByName(sv.Type(), srcField)
	}
	v := reflect.MakeSlice(dt, n, n)
	for i := 0; i < n; i++ {
		v.Index(i).Set(ptrTo(sv, i, sf).Elem())
	}
	for i, j, k := 0, 0, max; i < g; i++ {
		if k > n {
			k = n
		}
		dv.Index(i).Field(df).Set(v.Slice3(j, k, k))
		j, k = k, k+max
	}
}

// Strings extracts string values from a slice of structs or struct pointers
// containing the named field.
func (ctx *Ctx) Strings(src interface{}, srcField string) []string {
	sv, n := sliceValue(src)
	if n == 0 {
		return nil
	}
	sf := -1
	if srcField != "" {
		sf, _ = fieldByName(sv.Type(), srcField)
	}
	out := make([]string, n)
	for i := range out {
		switch v := ptrTo(sv, i, sf).Elem(); v.Kind() {
		case reflect.String:
			out[i] = v.String()
		case reflect.Int64:
			out[i] = strconv.FormatInt(v.Int(), 10)
		case reflect.Invalid:
			// nil pointer
		default:
			panic("scan: unsupported field type: " + v.Type().String())
		}
	}
	return out
}

// output is an interface implemented by all SDK Output structs.
type output interface {
	SDKResponseMetadata() aws.Response
}

// Input returns the Input struct that generated the given Output struct.
func (*Ctx) Input(out output) interface{} {
	return out.SDKResponseMetadata().Request.Params
}

// CopyInput copies the value of an Input struct field that generated out into a
// new Input slice. It is equivalent to:
//
//	v := ctx.Input(out).(*T).Field
//	for i := range dst {
//		dst[i].Field = v
//	}
func (*Ctx) CopyInput(dst interface{}, field string, out output) {
	if dv, n := sliceValue(dst); n > 0 {
		df, _ := fieldByName(dv.Type(), field)
		in := out.SDKResponseMetadata().Request.Params
		v := reflect.ValueOf(in).Elem().FieldByName(field)
		for i := 0; i < n; i++ {
			dv.Index(i).Field(df).Set(v)
		}
	}
}

// MakeResources adds new resources to ctx.Resources. See MakeResources method
// of tfx.ProviderMap for more info.
func (ctx *Ctx) MakeResources(typ string, attrs tfx.AttrGen) error {
	return ctx.addResources(tfx.Providers.MakeResources(typ, attrs))
}

// ImportResources adds new resources to ctx.Resources. See ImportResources
// method of tfx.ProviderMap for more info.
func (ctx *Ctx) ImportResources(typ string, attrs tfx.AttrGen) error {
	return ctx.addResources(tfx.Providers.ImportResources(typ, attrs))
}

// tryNext tries to run all links that depend on api.
func (ctx *Ctx) tryNext(api string) {
	if api != "" && ctx.Mode(RootsOnly) {
		return
	}
	for _, next := range ctx.svc.next[api] {
		for _, lnk := range ctx.svc.api[next] {
			ctx.tryRun(lnk)
		}
	}
}

// tryRun creates a new call batch for lnk if all dependencies are satisfied.
func (ctx *Ctx) tryRun(lnk *link) {
	if ctx.Calls[lnk.api] != nil || ctx.run[lnk] != nil {
		return // Finished or already running
	}
	for _, dep := range lnk.deps {
		if ctx.Calls[dep] == nil {
			return // Waiting for dependency
		}
	}

	// Allocate new batch instance
	b := &batch{ctx: ctx, lnk: lnk}
	ctx.run[lnk] = b
	defer func(b *batch) {
		if len(b.all) > 0 {
			b.next = append(b.next, b.all...)
			b.ctx.readyCalls += len(b.all)
		} else {
			b.ctx.finish(b)
		}
	}(b)
	if ctx.Mode(TFState) && !lnk.postProc {
		return // Link not needed for output post-processing
	}

	// Extract all dependencies from ctx.out
	type src struct {
		call *Call
		idx  int
		out  reflect.Value
	}
	outs := make([][]src, 1+len(lnk.deps))
	outs[0] = []src{{out: reflect.ValueOf(ctx.iface)}}
	for i, dep := range lnk.deps {
		calls := ctx.Calls[dep]
		srcs := make([]src, 0, len(calls))
		for _, c := range calls {
			for i, out := range c.Out {
				srcs = append(srcs, src{c, i, reflect.ValueOf(out)})
			}
		}
		if len(srcs) == 0 {
			return
		}
		outs[i+1] = srcs
	}

	// Invoke input method for every combination of outs (cartesian product)
	idx := make([]int, len(outs))
	args := make([]reflect.Value, len(outs))
	for {
		for i, j := range idx {
			args[i] = outs[i][j].out
		}
		inputs := lnk.input.Call(args)[0]
		if n := inputs.Len(); n > 0 {
			var src map[string]int
			if len(idx) > 1 {
				src = make(map[string]int, len(idx)-1)
				for i, j := range idx[1:] {
					s := &outs[i+1][j]
					src[s.call.ID] = s.idx
				}
			}
			calls := make([]Call, n)
			var stats []Stats
			if ctx.Mode(KeepStats) {
				stats = make([]Stats, n)
			}
			for i := range calls {
				c := &calls[i]
				if stats != nil {
					c.Stats = &stats[i]
				}
				c.Src = src
				c.In = inputs.Index(i).Addr().Interface()
				c.bat = b
				b.all = append(b.all, c)
			}
		} else if len(lnk.deps) == 0 {
			panic("scan: no inputs for root API: " +
				ctx.svc.name + ":" + lnk.api)
		}
		for i := len(outs) - 1; ; i-- {
			if idx[i]++; idx[i] < len(outs[i]) {
				break
			}
			if idx[i] = 0; i == 0 {
				return
			}
		}
	}
}

// next returns the next call to execute.
func (ctx *Ctx) next() *Call {
	if ctx.readyCalls == 0 {
		return nil
	}
	for _, bat := range ctx.run {
		if n := len(bat.next); n > 0 {
			c := bat.next[n-1]
			bat.next = bat.next[:n-1]
			bat.wait++
			ctx.totalCalls++
			ctx.readyCalls--
			return c
		}
	}
	panic("scan: inconsistent context state")
}

// done updates context state after call completion and returns true when the
// entire service has been scanned.
func (ctx *Ctx) done(c *Call) bool {
	if c.Err != nil && c.Err.Code != "" && len(c.Out) == 0 {
		ctx.iface.HandleError(c.req, c.Err)
	}
	ctx.postProcess(c)
	b := c.bat
	c.bat = nil
	c.req = nil
	if b.wait--; b.done() {
		ctx.finish(b)
	}
	return len(ctx.run) == 0
}

// postProcess calls post-processing service methods on all call outputs.
func (ctx *Ctx) postProcess(c *Call) {
	if !ctx.Mode(TFState) || len(c.Out) == 0 || !c.bat.lnk.postProc {
		return
	}
	fn := ctx.svc.postProc[reflect.TypeOf(c.Out[0])]
	if !fn.IsValid() {
		return
	}
	args := []reflect.Value{reflect.ValueOf(ctx.iface), {}}
	for _, out := range c.Out {
		args[1] = reflect.ValueOf(out)
		if err := fn.Call(args)[0]; !err.IsNil() {
			// TODO: Pass up to scanner and terminate scan?
			err := err.Interface().(error)
			panic("scan: service tfstate error: " + err.Error())
		}
	}
}

// finish combines calls from related batches into ctx.out.
func (ctx *Ctx) finish(b *batch) {
	n, api := 0, b.lnk.api
	for _, rel := range ctx.svc.api[api] {
		b := ctx.run[rel]
		if b == nil || !b.done() {
			return
		}
		n += len(b.all)
	}
	out := make([]*Call, 0, n)
	for _, rel := range ctx.svc.api[api] {
		out = append(out, ctx.run[rel].all...)
		delete(ctx.run, rel)
	}
	ctx.Calls[api] = out
	ctx.tryNext(api)
}

// addResources adds new resources to ctx.Resources.
func (ctx *Ctx) addResources(rs []tfx.Resource, err error) error {
	if len(rs) == 0 || err != nil {
		return err
	}
	if ctx.Resources == nil {
		ctx.Resources = make(map[string]*tf.ResourceState, len(rs))
	}
	for _, r := range rs {
		// TODO: Update key with region
		if _, dup := ctx.Resources[r.Key]; dup {
			panic("scan: resource state key collision: " + r.Key)
		}
		ctx.Resources[r.Key] = r.ResourceState
	}
	return nil
}

// batch contains all calls for one link.
type batch struct {
	ctx  *Ctx    // Parent context
	lnk  *link   // Link metadata
	all  []*Call // Call for each input
	next []*Call // Calls waiting to be executed
	wait int     // Number of calls currently being executed
}

// done returns true when all calls in the batch have been executed.
func (b *batch) done() bool { return len(b.next) == 0 && b.wait == 0 }

// scanner executes calls for multiple contexts until the entire scan is done.
// It implements heap.Interface for context scheduling.
type scanner struct {
	heap  []*Ctx       // Active context heap
	idx   map[*Ctx]int // Ctx index in heap
	ech   chan<- *Call // Execution channel
	rch   <-chan *Call // Return channel
	calls int          // Call counter
}

// newScanner starts worker goroutines in preparation for scanning contexts.
func newScanner(all []*Ctx, workers int) scanner {
	run := make([]*Ctx, 0, len(all))
	idx := make(map[*Ctx]int, len(all))
	for _, ctx := range all {
		if ctx.tryNext(""); len(ctx.run) > 0 {
			idx[ctx] = len(run)
			run = append(run, ctx)
		}
	}
	if len(run) == 0 {
		return scanner{}
	}
	if workers < 1 {
		workers = 64
	}
	// Channels must be unbuffered to ensure accurate stats
	ech := make(chan *Call)
	rch := make(chan *Call)
	for ; workers > 0; workers-- {
		go func(ech <-chan *Call, rch chan<- *Call) {
			var b bytes.Buffer
			b.Grow(512)
			j := json.NewEncoder(&b)
			j.SetEscapeHTML(false)
			h := sha512.New512_256()
			for c := range ech {
				if c.ID == "" {
					c.ID = c.id(&b, j, h)
					b.Reset()
					h.Reset()
				}
				c.exec()
				rch <- c
			}
		}(ech, rch)
	}
	return scanner{run, idx, ech, rch, 0}
}

// scan scans all active contexts until the heap is empty.
func (s *scanner) scan() {
	if len(s.heap) == 0 {
		return
	}
	defer close(s.ech)
	heap.Init(s)
	next := s.next()
	for {
		if next != nil {
			select {
			case s.ech <- next:
				next.Stats.exec()
			case c := <-s.rch:
				if s.done(c) {
					return
				}
				continue
			}
		} else if s.done(<-s.rch) {
			return
		}
		next = s.next()
	}
}

// next returns the next call to execute.
func (s *scanner) next() *Call {
	c := s.heap[0].next()
	if c != nil {
		c.Stats.ready(s.calls)
		s.calls++
		heap.Fix(s, 0)
	}
	return c
}

// done updates scan state after call completion and returns true when the scan
// is done.
func (s *scanner) done(c *Call) bool {
	updateTypes(c.req)
	if ctx := c.bat.ctx; ctx.done(c) {
		heap.Remove(s, s.idx[ctx])
	} else {
		heap.Fix(s, s.idx[ctx])
	}
	c.Stats.done(c.Err)
	return len(s.heap) == 0
}

// Len returns the number of active contexts.
func (s *scanner) Len() int {
	return len(s.heap)
}

// Less returns true if the context at index i has higher priority than the
// context at index j.
func (s *scanner) Less(i, j int) bool {
	ci, cj := s.heap[i], s.heap[j]
	if ri, rj := ci.readyCalls > 0, cj.readyCalls > 0; ri != rj {
		return ri
	}
	// TODO: Consider API graph, try to satisfy dependencies
	return ci.totalCalls < cj.totalCalls
}

// Swap swaps two contexts in the heap.
func (s *scanner) Swap(i, j int) {
	ci, cj := s.heap[i], s.heap[j]
	s.idx[ci], s.idx[cj] = j, i
	s.heap[i], s.heap[j] = cj, ci
}

// Push satisfies heap.Interface, but is unused.
func (s *scanner) Push(x interface{}) {
	panic("scan: heap.Push() called on scanner")
}

// Pop removes the last context from the heap.
func (s *scanner) Pop() interface{} {
	i := len(s.heap) - 1
	v := s.heap[i]
	s.heap = s.heap[:i]
	delete(s.idx, v)
	return v
}

// sliceValue verifies that s is a slice type, and returns its value and length.
func sliceValue(s interface{}) (reflect.Value, int) {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Slice {
		return v, v.Len()
	}
	panic("scan: not a slice: " + v.Type().String())
}

// ensureSliceAt verifies the length of slice pointer value in p or allocates a
// new slice if *p is nil. It returns the slice value.
func ensureSliceAt(p interface{}, n int) reflect.Value {
	v := reflect.ValueOf(p).Elem()
	if v.Kind() == reflect.Slice {
		if v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), n, n))
		} else if v.Len() != n {
			panic(fmt.Sprintf("scan: len mismatch: want=%d have=%d", n, v.Len()))
		}
		return v
	}
	panic("scan: not a slice: " + v.Type().String())
}

// fieldByName returns the index and type of the named field in t, which may be
// a struct pointer or container type.
func fieldByName(t reflect.Type, name string) (int, reflect.Type) {
	for t.Kind() != reflect.Struct {
		t = t.Elem()
	}
	f, ok := t.FieldByName(name)
	if !ok || len(f.Index) != 1 {
		panic("scan: field not found: " + t.String() + "." + name)
	}
	return f.Index[0], f.Type
}

// ptrTo returns a pointer to the value at v[i] or v[i].f if f is non-negative.
// No additional indirection is made if the value is already a pointer.
func ptrTo(v reflect.Value, i, f int) reflect.Value {
	if v = v.Index(i); f >= 0 {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		v = v.Field(f)
	}
	if v.Kind() != reflect.Ptr {
		v = v.Addr()
	}
	return v
}
