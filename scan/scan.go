package scan

import (
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var svcRegistry = make(map[string]service)

func Register(s Service) {
	name := path.Base(reflect.TypeOf(s.NewFunc()).Out(0).Elem().PkgPath())
	if _, ok := svcRegistry[name]; ok {
		panic("scan: service already registered: " + name)
	}
	if svcRegions[name] == nil {
		r := svcRegions[s.Name()]
		if r == nil {
			panic("scan: invalid service name: " + name)
		}
		svcRegions[name] = r
	}
	svcRegistry[name] = service{Service: s, api: apiMap(s)}
}

func Services() []string {
	all := make([]string, 0, len(svcRegistry))
	for name := range svcRegistry {
		all = append(all, name)
	}
	sort.Strings(all)
	return all
}

type Service interface {
	Name() string
	NewFunc() interface{}
	Roots() []interface{}
}

type Map map[string][]*State

func Scan(services []string, configs []*aws.Config, workers int) Map {
	cfgs := testCfgs(configs)
	for i := range cfgs {
		if cfgs[i].err != nil {
			panic(fmt.Sprintf("scan: invalid config at index %d (%v)",
				i, cfgs[i].err))
		}
	}
	m := newMap(services, len(cfgs))
	if len(m) == 0 {
		return m
	}
	send := startWorkers(workers)
	defer close(send)
	var wg sync.WaitGroup
	for svc, states := range m {
		for _, cfg := range cfgs {
			if !canScan(svc, cfg.Region) {
				continue
			}
			s := newState(svc, cfg, send)
			states = append(states, s)
			wg.Add(1)
			go func() {
				defer wg.Done()
				s.scan()
			}()
		}
		m[svc] = states
	}
	wg.Wait()
	return m
}

func newMap(services []string, nCfgs int) Map {
	if len(services) == 0 {
		m := make(Map, len(svcRegistry))
		for name := range svcRegistry {
			m[name] = make([]*State, 0, nCfgs)
		}
		return m
	}
	m := make(Map, len(services))
	for _, name := range services {
		if _, valid := svcRegistry[name]; !valid {
			panic("scan: unknown service: " + name)
		}
		m[name] = make([]*State, 0, nCfgs)
	}
	return m
}

func (m Map) OmitEmpty() {
	clean := func(val interface{}) interface{} {
		if val == nil {
			return val
		}
		t := reflect.TypeOf(val).Elem()
		if t.Kind() != reflect.Struct {
			return val
		}
		v := reflect.ValueOf(val)
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
		m := make(map[string]interface{}, v.NumField())
		for i := t.NumField() - 1; i >= 0; i-- {
			if fv, name := v.Field(i), t.Field(i).Name; keepField(fv, name) {
				m[name] = fv.Interface()
			}
		}
		if len(m) == 0 {
			return nil
		}
		return m
	}
	for svc, states := range m {
		keep := states[:0]
		for _, state := range states {
			haveResults := false
			for api, results := range state.Data {
				if len(results) == 0 {
					delete(state.Data, api)
					continue
				}
				haveResults = true
				for i := range results {
					r := &results[i]
					r.Input = clean(r.Input)
					r.Output = clean(r.Output)
				}
			}
			if haveResults {
				keep = append(keep, state)
			}
		}
		m[svc] = keep
	}
}

type Result struct {
	Input  interface{} `json:"input,omitempty"`
	Output interface{} `json:"output,omitempty"`
	Error  *string     `json:"error,omitempty"`
}

type State struct {
	Account string              `json:"account"`
	Region  string              `json:"region"`
	Data    map[string][]Result `json:"data"`

	svc    reflect.Value
	client reflect.Value
	api    map[string]*api
	send   chan<- *call
	wg     sync.WaitGroup
}

func newState(service string, cfg *cfg, send chan<- *call) *State {
	svc := svcRegistry[service]
	svcNew := reflect.ValueOf(svc.NewFunc())
	in := []reflect.Value{reflect.ValueOf(*cfg.Config)}
	s := &State{
		Account: cfg.account,
		Region:  cfg.Region,
		Data:    make(map[string][]Result, len(svc.api)),
		svc:     reflect.New(reflect.TypeOf(svc.Service)),
		client:  svcNew.Call(in)[0],
		api:     svc.api,
		send:    send,
	}
	if f := s.svc.Elem().FieldByName("State"); f.IsValid() {
		f.Set(reflect.ValueOf(s))
	}
	s.svc = s.svc.Elem()
	return s
}

func (s *State) scan() {
	var next []string
	var calls []call
	for {
		next = next[:0]
		for name := range s.api {
			if s.canCall(name) {
				next = append(next, name)
			}
		}
		if len(next) == 0 {
			if len(s.Data) != len(s.api) {
				// Unable to make progress
				// TODO: Should be caught in apiMap
				panic("scan: service deadlock")
			}
			return
		}
		sort.Strings(next)
		calls = calls[:0]
		for _, name := range next {
			n := len(calls)
			calls = s.append(calls, s.api[name])
			// TODO: Need to combine existing results (e.g. tags)
			s.Data[name] = make([]Result, 0, len(calls)-n)
		}
		s.exec(calls)
	}
}

// canCall returns true if the API was not already called and all of its
// dependencies are satisfied.
func (s *State) canCall(api string) bool {
	// TODO: Replace with fixed call order in apiMap
	if _, done := s.Data[api]; done {
		return false
	}
	for _, dep := range s.api[api].deps {
		if _, ok := s.Data[dep]; !ok {
			return false
		}
	}
	return true
}

func (s *State) append(calls []call, api *api) []call {
	args := make([][]reflect.Value, api.input.Type().NumIn())
	args[0] = []reflect.Value{s.svc}
	for i, dep := range api.deps {
		outs := make([]reflect.Value, 0, len(s.Data[dep]))
		for _, result := range s.Data[dep] {
			if result.Error == nil {
				outs = append(outs, reflect.ValueOf(result.Output))
			}
		}
		if len(outs) == 0 {
			return calls
		}
		args[i+1] = outs
	}

	// Cartesian product of args
	idx := make([]int, len(args))
	in := make([]reflect.Value, len(args))
	for {
		for i, j := range idx {
			in[i] = args[i][j]
		}
		inputs := api.input.Call(in)[0]
		if inputs.Kind() == reflect.Interface {
			inputs = inputs.Elem() // Root func
		}
		for i, n := 0, inputs.Len(); i < n; i++ {
			calls = append(calls, call{
				WaitGroup: &s.wg,
				api:       api,
				client:    s.client,
				input:     inputs.Index(i).Interface(),
			})
		}
		for i := len(args) - 1; ; i-- {
			if idx[i]++; idx[i] < len(args[i]) {
				break
			}
			if idx[i] = 0; i == 0 {
				return calls
			}
		}
	}
}

// TODO: Have service specify which errors to ignore

var ignoreErrors = map[string]bool{
	"AccessDenied":                 true, // s3
	"AuthorizationHeaderMalformed": true, // s3
	"BucketRegionError":            true, // s3
	"NoSuchBucketPolicy":           true, // s3
	"NoSuchCORSConfiguration":      true, // s3
	"NoSuchEntity":                 true, // iam.GetLoginProfile
	"NoSuchTagSet":                 true, // s3
	"NoSuchWebsiteConfiguration":   true, // s3
}

func (s *State) exec(calls []call) {
	s.wg.Add(len(calls))
	for i := range calls {
		s.send <- &calls[i]
	}
	s.wg.Wait()
	for i := range calls {
		c := &calls[i]
		if c.err == nil && noOutput(c.output) {
			continue
		}
		var err *string
		if c.err != nil {
			if e, ok := c.err.(awserr.Error); ok && ignoreErrors[e.Code()] {
				continue
			}
			err = aws.String(c.err.Error())
		}
		s.Data[c.api.name] = append(s.Data[c.api.name], Result{
			Input:  c.input,
			Output: c.output,
			Error:  err,
		})
	}
}

type cfg struct {
	*aws.Config
	account string
	err     error
}

func testCfgs(cfgs []*aws.Config) []*cfg {
	out := make([]*cfg, len(cfgs))
	keys := make(map[string][]*cfg)
	for i := range cfgs {
		c := &cfg{Config: cfgs[i]}
		creds, err := c.Credentials.Retrieve()
		if err == nil {
			keys[creds.AccessKeyID] = append(keys[creds.AccessKeyID], c)
		} else {
			c.err = err
		}
		out[i] = c
	}
	// Call GetCallerIdentity for each unique access key
	var wg sync.WaitGroup
	wg.Add(len(keys))
	for _, cfgs := range keys {
		go func(cfgs []*cfg) {
			defer wg.Done()
			client := sts.New(*cfgs[0].Config)
			id, err := client.GetCallerIdentityRequest(nil).Send()
			if err == nil {
				account := aws.StringValue(id.Account)
				for _, c := range cfgs {
					c.account = account
				}
			} else {
				for _, c := range cfgs {
					c.err = err
				}
			}
		}(cfgs)
	}
	wg.Wait()
	return out
}

type service struct {
	Service
	api map[string]*api
}

type api struct {
	name  string        // API name
	deps  []string      // APIs that must be called first
	input reflect.Value // Service method to get input
	req   reflect.Value // Client method to create request
	send  reflect.Value // Request method to execute call
}

var apiBlacklist = map[string]bool{
	"ResumeProcessesRequest":       true, // autoscaling
	"GetBucketNotificationRequest": true, // s3
}

func apiMap(s Service) map[string]*api {
	// Create a map of input types to request constructors
	client := reflect.TypeOf(s.NewFunc()).Out(0)
	reqFuncs := make(map[reflect.Type]reflect.Method, client.NumMethod())
	for i := client.NumMethod() - 1; i >= 0; i-- {
		m := client.Method(i)
		if strings.HasSuffix(m.Name, "Request") && !apiBlacklist[m.Name] {
			in := m.Type.In(1) // *Input
			if _, mapped := reqFuncs[in]; mapped {
				panic("scan: input already mapped: " + m.Name)
			}
			reqFuncs[in] = m
		}
	}
	apiFor := func(inputPtr reflect.Type) *api {
		req, ok := reqFuncs[inputPtr]
		if !ok {
			panic("scan: no request method for: " + inputPtr.String())
		}
		send, ok := req.Type.Out(0).MethodByName("Send")
		if !ok {
			panic("scan: no send method for: " + req.Type.Out(0).Name())
		}
		return &api{
			name: strings.TrimSuffix(req.Name, "Request"),
			req:  req.Func,
			send: send.Func,
		}
	}

	// Create service API map
	iface := reflect.TypeOf(&s).Elem()
	ignore := make(map[string]bool, iface.NumMethod())
	for i := iface.NumMethod() - 1; i >= 0; i-- {
		ignore[iface.Method(i).Name] = true
	}
	svc := reflect.TypeOf(s)
	roots := s.Roots()
	apis := make(map[string]*api, len(roots)+svc.NumMethod()-len(ignore))
	for _, root := range roots {
		inputs := root
		api := apiFor(reflect.TypeOf(root).Elem())
		api.input = reflect.ValueOf(func(svc interface{}) interface{} {
			return inputs
		})
		apis[api.name] = api
	}
	for i := svc.NumMethod() - 1; i >= 0; i-- {
		m := svc.Method(i)
		if ignore[m.Name] {
			continue
		}
		api := apiFor(m.Type.Out(0).Elem())
		if api.name != m.Name {
			panic("scan: invalid api name: " + m.Name)
		}
		if n := m.Type.NumIn() - 1; n > 0 {
			api.deps = make([]string, n)
			for i := range api.deps {
				name := m.Type.In(i + 1).Elem().Name()
				if !strings.HasSuffix(name, "Output") {
					panic("scan: invalid input: " + name)
				}
				api.deps[i] = strings.TrimSuffix(name, "Output")
			}
		}
		api.input = m.Func
		apis[api.name] = api
	}

	// TODO: Validate dependencies, ensure no cycles, etc.
	return apis
}

func (api *api) call(client reflect.Value, input interface{}) (interface{}, error) {
	in := []reflect.Value{client, reflect.ValueOf(input)}
	req := api.req.Call(in)[0]
	out := api.send.Call(append(in[:0], req))
	// TODO: Pagination
	if out[1].IsNil() {
		return out[0].Interface(), nil
	}
	return nil, out[1].Interface().(error)
}

type call struct {
	*sync.WaitGroup
	api    *api
	client reflect.Value
	input  interface{}
	output interface{}
	err    error
}

func startWorkers(n int) chan<- *call {
	if n < 1 {
		n = 100
	}
	ch := make(chan *call, n)
	for ; n > 0; n-- {
		go func() {
			for c := range ch {
				c.output, c.err = c.api.call(c.client, c.input)
				c.Done()
			}
		}()
	}
	return ch
}

func noOutput(out interface{}) bool {
	t := reflect.TypeOf(out).Elem()
	v := reflect.ValueOf(out).Elem()
	for i := t.NumField() - 1; i >= 0; i-- {
		if keepField(v.Field(i), t.Field(i).Name) {
			return false
		}
	}
	return true
}

func keepField(v reflect.Value, name string) bool {
	return v.CanSet() && (hasValue(v) || name == "LocationConstraint") &&
		name != "IsTruncated" && name != "NextToken" && name != "MaxItems"
}

func hasValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr:
		return !v.IsNil() && hasValue(v.Elem())
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() > 0
	case reflect.Chan, reflect.Func, reflect.Interface:
		return !v.IsNil()
	}
	return true
}
