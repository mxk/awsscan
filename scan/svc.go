package scan

import (
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
)

var svcRegistry registry

// Register adds a new scannable service to the registry.
func Register(id string, newFunc interface{}, iface svcIface, roots ...interface{}) struct{} {
	name := reflect.TypeOf(newFunc).Out(0).String()
	name = name[1:strings.IndexByte(name, '.')]
	svcRegistry.register(name, id, newFunc, iface, roots)
	return struct{}{}
}

// ServiceNames returns the names of all scannable services.
func ServiceNames() []string {
	all := make([]string, 0, len(svcRegistry.reg))
	for name := range svcRegistry.reg {
		all = append(all, name)
	}
	sort.Strings(all)
	return all
}

// Service contains information about one registered service.
type Service struct {
	ID      string        // Endpoint ID
	NewFunc interface{}   // SDK client constructor
	Iface   interface{}   // Scanner implementation
	Roots   []interface{} // Root inputs
}

// ServiceInfo returns information about one registered service.
func ServiceInfo(name string) Service {
	if svc := svcRegistry.reg[name]; svc != nil {
		return Service{
			ID:      svc.id,
			NewFunc: svc.newClient.Interface(),
			Iface:   reflect.New(svc.typ).Elem().Interface(),
			Roots:   append([]interface{}{}, svc.roots...),
		}
	}
	return Service{}
}

// API returns a map of all supported APIs and their dependencies for the
// specified service.
func API(service string) map[string][]string {
	svc := svcRegistry.get()[service]
	if svc == nil {
		return nil
	}
	api := make(map[string][]string, len(svc.api))
	depSet := make(map[string]struct{})
	for name, links := range svc.api {
		for _, lnk := range links {
			for _, dep := range lnk.deps {
				depSet[dep] = struct{}{}
			}
		}
		if len(depSet) == 0 {
			api[name] = nil
			continue
		}
		deps := make([]string, 0, len(depSet))
		for dep := range depSet {
			deps = append(deps, dep)
			delete(depSet, dep)
		}
		sort.Strings(deps)
		api[name] = deps
	}
	return api
}

// registry contains all registered services.
type registry struct {
	once sync.Once
	reg  map[string]*svc
}

// svcIface must be implemented by all services. The base implementation is
// provided by *Ctx.
type svcIface interface {
	UpdateRequest(req *aws.Request)
	HandleError(req *aws.Request, err *Err)
}

// svc describes a scannable service.
type svc struct {
	name      string                         // Unique service name (client package name)
	id        string                         // Endpoint ID for region check
	newClient reflect.Value                  // func New(aws.Config) *T
	typ       reflect.Type                   // Type implementing svcIface
	roots     []interface{}                  // Root API inputs
	links     []*link                        // Link for each service root and method
	api       map[string][]*link             // API name index
	next      map[string][]string            // API call graph (key called before values)
	postProc  map[reflect.Type]reflect.Value // Output post-processing methods
}

// register adds a new scannable service to the registry.
func (r *registry) register(name, id string, newFunc interface{}, iface svcIface, roots []interface{}) {
	if r.reg[name] != nil {
		panic("scan: service already registered: " + name)
	} else if r.reg == nil {
		r.reg = make(map[string]*svc)
	}
	r.reg[name] = &svc{
		name:      name,
		id:        id,
		newClient: reflect.ValueOf(newFunc),
		typ:       reflect.TypeOf(iface),
		roots:     roots,
	}
}

// get returns service registry after a one-time initialization.
func (r *registry) get() map[string]*svc {
	r.once.Do(func() {
		// Ctx methods are ignored
		t := reflect.TypeOf((*Ctx)(nil))
		n := t.NumMethod()
		ctxMethod := make(map[string]bool, n)
		for i := n - 1; i >= 0; i-- {
			ctxMethod[t.Method(i).Name] = true
		}
		for _, s := range r.reg {
			s.init(ctxMethod)
		}
	})
	return r.reg
}

// link associates a service method that returns zero or more inputs with a
// client method that returns a request for each input. It is a node in a
// directed acyclic graph, which defines the order of API calls. One API, such
// as ListTagsForResource, may be referenced by multiple links, so there is an
// N:1 relationship between service methods and client methods.
type link struct {
	api      string        // API name
	deps     []string      // APIs that must be called first (input method args)
	input    reflect.Value // Service method to get input
	req      reflect.Value // Client method to create request
	postProc bool          // Is this link needed for post-processing?
}

func (s *svc) init(ctxMethod map[string]bool) {
	// Identify link and output post-processing methods
	s.postProc = make(map[reflect.Type]reflect.Value)
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	var isLink bitSet
	for i := s.typ.NumMethod() - 1; i >= 0; i-- {
		m := s.typ.Method(i)
		if ctxMethod[m.Name] {
			continue
		}
		if m.Type.NumIn() == 2 && m.Type.NumOut() == 1 &&
			m.Type.Out(0) == errorType {
			out := m.Type.In(1)
			if _, dup := s.postProc[out]; dup {
				panic("scan: multiple post-process methods: " + out.String())
			}
			s.postProc[out] = m.Func
		} else {
			isLink.set(i)
		}
	}

	// Create links for each service root and method
	linkPool := make([]link, len(s.roots)+isLink.len())
	newLink := func(fn reflect.Value) {
		lnk := &linkPool[len(s.links)]
		lnk.api = apiName(fn.Type().Out(0))
		lnk.input = fn
		s.links = append(s.links, lnk)
		s.api[lnk.api] = append(s.api[lnk.api], lnk)
	}
	s.links = make([]*link, 0, len(linkPool))
	s.api = make(map[string][]*link, len(linkPool))
	s.next = make(map[string][]string, len(linkPool))
	for i := range s.roots {
		v := reflect.ValueOf(s.roots[i]) // []Input
		t := v.Type()
		if v.Len() == 0 {
			v = reflect.MakeSlice(t, 1, 1)
		}
		t = reflect.FuncOf([]reflect.Type{s.typ}, []reflect.Type{t}, false)
		q := []reflect.Value{v}
		newLink(reflect.MakeFunc(t, func([]reflect.Value) []reflect.Value {
			return q
		}))
	}
	for i, n := 0, s.typ.NumMethod(); i < n; i++ {
		if isLink.test(i) {
			newLink(s.typ.Method(i).Func)
		}
	}

	// Find client request method for each API name and create output type map
	client := s.newClient.Type().Out(0)
	outMap := make(map[reflect.Type]string, len(s.api))
	for api, links := range s.api {
		req := getMethod(client, api+"Request")
		send := getMethod(req.Type.Out(0), "Send")
		out := send.Type.Out(0)
		if outMap[out] != "" {
			panic("scan: output type collision: " + out.String())
		} else if out.Elem().NumField() > maxFields {
			panic("scan: typeBitSet overflow: " + out.String())
		}
		outMap[out] = api
		postProc := s.postProc[out].IsValid()
		for _, lnk := range links {
			lnk.req = req.Func
			lnk.postProc = postProc
		}
	}
	for out := range s.postProc {
		if outMap[out] == "" {
			panic("scan: unsatisfied post-process method input: " +
				out.String())
		}
	}

	// Find link dependencies
	for _, lnk := range s.links {
		fn := lnk.input.Type() // func(svcIface, *AbcOutput, ...) []*XyzInput
		if n := fn.NumIn() - 1; n > 0 {
			lnk.deps = make([]string, n)
			for i := range lnk.deps {
				out := fn.In(i + 1)
				var ok bool
				if lnk.deps[i], ok = outMap[out]; !ok {
					panic("scan: unsatisfied dependency: " +
						out.String() + " -> " + fn.Out(0).String())
				}
				if lnk.postProc {
					for _, dep := range s.api[lnk.deps[i]] {
						dep.postProc = true
					}
				}
			}
		}
	}

	// Create call graph and test for dependency cycles
	called := make(map[string]bool, len(s.api))
	tryCall := func(api string) {
		links := s.api[api]
		for _, lnk := range links {
			for _, dep := range lnk.deps {
				if !called[dep] {
					return
				}
			}
		}
		called[api] = true
		for _, lnk := range links {
			deps := lnk.deps
			if deps == nil {
				deps = []string{""}
			}
			for _, dep := range deps {
				next := s.next[dep]
				if n := len(next); n == 0 || next[n-1] != api {
					s.next[dep] = append(next, api)
				}
			}
		}
	}
	apis := make([]string, 0, len(s.api))
	for api := range s.api {
		apis = append(apis, api)
	}
	sort.Strings(apis)
	for len(called) < len(apis) {
		n := len(called)
		for _, api := range apis {
			if !called[api] {
				tryCall(api)
			}
		}
		if len(called) == n {
			var cycle strings.Builder
			for _, api := range apis {
				if !called[api] {
					if cycle.Len() != 0 {
						cycle.WriteByte(',')
					}
					cycle.WriteString(api)
				}
			}
			panic("scan: " + s.name + " dependency cycle: " + cycle.String())
		}
	}
}

// apiName extracts service API name from []XyzInput type.
func apiName(q reflect.Type) string {
	if q.Kind() != reflect.Slice {
		panic("scan: not a slice: " + q.String())
	} else if q = q.Elem(); q.Kind() != reflect.Struct {
		panic("scan: not a struct: " + q.String())
	} else if q.NumField() > maxFields {
		panic("scan: typeBitSet overflow: " + q.String())
	}
	name := q.Name()
	api := strings.TrimSuffix(name, "Input")
	if api == "" || len(api) == len(name) {
		panic("scan: not input: " + q.String())
	}
	return api
}

// getMethod returns a method of t by name.
func getMethod(t reflect.Type, name string) reflect.Method {
	if m, ok := t.MethodByName(name); ok {
		return m
	}
	panic("scan: method not found: " + t.String() + "." + name)
}
