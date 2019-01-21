package svc

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mxk/cloudcover/awsscan/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// svcIface is copied from the scan package to avoid exporting it.
type svcIface interface {
	UpdateRequest(req *aws.Request)
	HandleError(req *aws.Request, err *scan.Err)
}

// mockOutput is an optional interface that allows services to customize mock
// Output structs during testing.
type mockOutput interface {
	svcIface
	mockOutput(out interface{})
}

func TestSvc(t *testing.T) {
	// Validate service names, ensure non-empty API graph
	names := scan.ServiceNames()
	all := make([]svcIface, len(names))
	for i, name := range names {
		s := scan.ServiceInfo(name).Iface.(svcIface)
		all[i] = s
		require.Equal(t, name+"Svc", reflect.TypeOf(s).Name())
		require.NotEmpty(t, scan.API(name))
	}

	// Call each method of each service in each scan mode. All methods must
	// return at least one Input struct in at least one mode.
	modes := []scan.Mode{0, scan.CloudAssert}
	ctx := reflect.TypeOf((*scan.Ctx)(nil))
	ctxMethod := make(map[string]bool)
	for i := ctx.NumMethod() - 1; i >= 0; i-- {
		ctxMethod[ctx.Method(i).Name] = true
	}
	err := scan.Err{
		Status:  404,
		Code:    "NotFound",
		Message: "Resource not found",
	}
	req := aws.Request{Operation: &aws.Operation{}}
	for _, svc := range all {
		s := newSvc(svc, ctxMethod)
		for req.Operation.Name = range s.inputs {
			s.iface.HandleError(&req, &err) // Nothing to test, just coverage
		}
		namePrefix := reflect.TypeOf(svc).Name() + "."
		for name, fn := range s.methods {
			api := apiName(fn.Type().Out(0).Elem())
			assert.True(t, strings.HasPrefix(name, namePrefix+api),
				"method %q does not begin with %q", name, api)
			var works bool
			for _, m := range modes {
				s.ctx.SetMode(m)
				out := fn.Call(s.getArgs(fn))
				works = works || out[0].Len() > 0
				// No early termination to test different branches
			}
			assert.True(t, works, "method %q does not work", name)
		}
	}
}

type svc struct {
	iface   svcIface
	ctx     *scan.Ctx
	methods map[string]reflect.Value
	inputs  map[string]reflect.Type
}

func newSvc(iface svcIface, ctxMethod map[string]bool) *svc {
	v := reflect.New(reflect.TypeOf(iface))
	s := &svc{
		iface:   v.Interface().(svcIface),
		ctx:     scan.TestCtx(iface),
		methods: make(map[string]reflect.Value),
		inputs:  make(map[string]reflect.Type),
	}
	v = v.Elem()
	v.FieldByName("Ctx").Set(reflect.ValueOf(s.ctx))
	t := v.Type()
	namePrefix := t.Name() + "."
	for i := v.NumMethod() - 1; i >= 0; i-- {
		// TODO: Make this filtering more robust
		if m := t.Method(i); !ctxMethod[m.Name] && m.Type.NumOut() == 1 &&
			m.Type.Out(0).Kind() == reflect.Slice {
			s.methods[namePrefix+m.Name] = v.Method(i)
			in := m.Type.Out(0).Elem()
			s.inputs[apiName(in)] = in
		}
	}
	return s
}

func (s *svc) getArgs(fn reflect.Value) []reflect.Value {
	t := fn.Type()
	args := make([]reflect.Value, t.NumIn())
	for i := range args {
		args[i] = reflect.New(t.In(i).Elem())
		s.mockOutput(args[i].Elem())
	}
	if iface, ok := s.iface.(mockOutput); ok {
		for i := range args {
			iface.mockOutput(args[i].Interface())
		}
	}
	return args
}

func (s *svc) mockOutput(v reflect.Value) {
	var mock func(v reflect.Value, depth int)
	mock = func(v reflect.Value, depth int) {
		if depth++; depth == 7 {
			return
		}
		switch t := v.Type(); v.Kind() {
		case reflect.Ptr:
			v.Set(reflect.New(t.Elem()))
			mock(v.Elem(), depth)
		case reflect.Slice:
			v.Set(reflect.MakeSlice(t, 1, 1))
			mock(v.Index(0), depth)
		case reflect.Struct:
			for i := v.NumField() - 1; i >= 0; i-- {
				if f := v.Field(i); f.CanSet() {
					mock(f, depth)
				}
			}
		}
	}
	// If the input type for this output is known (returned by another method,
	// not root), create a mock aws.Response for SDKResponseMetadata().
	if t, ok := s.inputs[apiName(v.Type())]; ok {
		in := reflect.New(t)
		rsp := (*aws.Response)(unsafe.Pointer(
			v.FieldByName("responseMetadata").UnsafeAddr()))
		rsp.Request = &aws.Request{Params: in.Interface()}
		mock(in.Elem(), 0)
	}
	mock(v, 0)
}

func apiName(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		panic("not a struct: " + t.String())
	}
	name := t.Name()
	if api := strings.TrimSuffix(name, "Input"); len(api) != len(name) {
		return api
	} else if api = strings.TrimSuffix(name, "Output"); len(api) != len(name) {
		return api
	}
	panic("not an input or output type: " + t.String())
}
