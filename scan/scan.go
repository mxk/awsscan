package scan

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	tf "github.com/hashicorp/terraform/terraform"
	"github.com/mxk/cloudcover/x/arn"
	"github.com/mxk/cloudcover/x/region"
	"github.com/mxk/cloudcover/x/tfx"
	"github.com/pkg/errors"
)

// Mode is a bitset that determines service scanning behavior. Scanners may
// issue different API calls in different modes.
type Mode uint32

const (
	RootsOnly   Mode = 1 << iota // Make only root API calls
	KeepStats                    // Maintain call statistics
	CloudAssert                  // Limit calls to those used by cloudassert
	TFState                      // Generate skeleton Terraform state
)

// Opts specifies optional scan parameters.
type Opts struct {
	Mode     Mode     // Scan mode
	Regions  []string // AWS regions
	Services []string // Service names
	Workers  int      // Maximum number of concurrent API calls
}

// Map contains all calls for one account/region/service, indexed by API name.
// Resources are indexed by Terraform state keys.
type Map struct {
	arn.Ctx
	Service   string
	Calls     map[string][]*Call
	Resources map[string]*tf.ResourceState
}

// Account creates a map of each service in each region using worker goroutines.
// If regions is empty, all regions within the current partition are scanned. If
// services is empty, all supported services are scanned.
func Account(cfg *aws.Config, op Opts) ([]*Map, error) {
	// Get account information
	id, err := ident(*cfg, op.Mode)
	if err != nil {
		return nil, err
	}
	ac := arn.Ctx{
		Partition: arn.Value(id.Arn).Partition(),
		Account:   aws.StringValue(id.Account),
	}

	// Filter out regions outside of the current partition
	if len(op.Regions) == 0 {
		// Filter out FIPS regions by default because of incorrect model data
		op.Regions = region.Related(ac.Partition)
		valid := op.Regions[:0]
		for _, r := range op.Regions {
			if !strings.HasPrefix(r, "fips-") {
				valid = append(valid, r)
			}
		}
		op.Regions = valid
	} else {
		valid := make([]string, 0, len(op.Regions))
		for _, r := range op.Regions {
			switch region.Partition(r) {
			case ac.Partition:
				valid = append(valid, r)
			case "":
				return nil, errors.Errorf("invalid region %q", r)
			}
		}
		op.Regions = valid
	}

	// Create Ctx for each valid region/service combination
	if len(op.Services) == 0 {
		op.Services = ServiceNames()
	}
	all := make([]*Ctx, 0, len(op.Services)*len(op.Regions))
	reg := svcRegistry.get()
	for _, s := range op.Services {
		svc := reg[s]
		if svc == nil {
			return nil, errors.Errorf("invalid or unsupported service %q", s)
		}
		for _, r := range op.Regions {
			if region.Supports(r, svc.id) {
				all = append(all, newCtx(cfg, ac.In(r), svc, op))
			}
		}
	}
	if len(all) == 0 {
		return nil, nil
	}

	// Scan and combine results
	s := newScanner(all, op.Workers)
	s.scan()
	m := make([]*Map, len(all))
	for i := range all {
		m[i] = &all[i].Map
	}
	return m, nil
}

// IO replaces SDK Input/Output struct types in Calls when a Map is compacted.
type IO map[string]interface{}

// WalkFunc is a function type called by Map.Walk().
type WalkFunc func(m *Map, api string, c *Call) error

// Walk calls fn for each map, API name, and call instance in maps. It returns
// the first non-nil error from fn.
func Walk(maps []*Map, fn WalkFunc) error {
	var apis []string
	for _, m := range maps {
		apis = apis[:0]
		for api := range m.Calls {
			apis = append(apis, api)
		}
		sort.Strings(apis)
		for _, api := range apis {
			for _, c := range m.Calls[api] {
				if err := fn(m, api, c); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Compact replaces all Input/Output structs in m with IO maps containing only
// non-zero values.
func Compact(maps []*Map) []*Map {
	skipFields := copySkipFields()
	keepMaps := maps[:0]
	for _, m := range maps {
		for api, calls := range m.Calls {
			keepCalls := calls[:0]
			for _, c := range calls {
				if c.Err != nil && c.Err.Ignore {
					continue
				}
				outs := c.Out[:0]
				for _, out := range c.Out {
					if v := compactIO(out, skipFields, false); v != nil {
						outs = append(outs, v)
					}
				}
				if len(outs) == 0 {
					if c.Err == nil {
						continue
					}
					outs = nil
				}
				c.Out = outs
				c.In = compactIO(c.In, skipFields, true)
				keepCalls = append(keepCalls, c)
			}
			if len(keepCalls) > 0 {
				m.Calls[api] = keepCalls
			} else {
				delete(m.Calls, api)
			}
		}
		if len(m.Calls) > 0 {
			keepMaps = append(keepMaps, m)
		}
	}
	return keepMaps
}

// NewTFState combines resources from all maps into a single Terraform state.
func NewTFState(maps []*Map) (*tf.State, error) {
	s := tfx.NewState()
	rs := s.RootModule().Resources
	for _, m := range maps {
		for k, r := range m.Resources {
			if _, dup := rs[k]; dup {
				return nil, fmt.Errorf("resource state key collision: %q", k)
			}
			rs[k] = r
		}
	}
	return s, nil
}

// ident returns caller identity for the current credentials, automatically
// detecting the correct partition when necessary.
func ident(cfg aws.Config, m Mode) (*sts.GetCallerIdentityOutput, error) {
	if m&CloudAssert == 0 {
		id, err := sts.New(cfg).GetCallerIdentityRequest(nil).Send()
		return id, errors.WithStack(err)
	}
	// This is here for compatibility with qa-harness, which doesn't set the
	// correct default region for GovCloud. Other clients are expected to
	// configure AWS environment variables correctly.
	var out, err atomic.Value
	var wg sync.WaitGroup
	ident := func(region string) {
		defer wg.Done()
		c := cfg
		c.Region = region
		if id, e := sts.New(c).GetCallerIdentityRequest(nil).Send(); e == nil {
			out.Store(id)
		} else {
			err.Store(errors.WithStack(e))
		}
	}
	regions := [...]string{"us-east-1", "us-gov-west-1"}
	wg.Add(len(regions))
	for _, r := range regions {
		go ident(r)
	}
	wg.Wait()
	if id := out.Load(); id != nil {
		return id.(*sts.GetCallerIdentityOutput), nil
	}
	return nil, err.Load().(error)
}

// compactIO converts an Input/Output struct pointer to an IO map, keeping only
// those fields that have valid data. It returns nil if all fields are empty.
func compactIO(io interface{}, skipFields typeBitSet, in bool) interface{} {
	v := reflect.ValueOf(io)
	if v.IsNil() {
		return nil
	}
	v = v.Elem()
	t := v.Type()
	sf := skipFields[t]
	skip := staticBitSet{sf[:]}
	var keep IO
	for i, n := 0, t.NumField(); i < n; i++ {
		if !skip.test(i) {
			if f := v.Field(i); keepValue(f, in) {
				if keep == nil {
					keep = make(IO, n-i)
				}
				keep[t.Field(i).Name] = f.Interface()
			}
		}
	}
	if len(keep) > 0 {
		return keep
	}
	return nil
}

// keepValue returns true if an Input/Output struct field has valid data.
func keepValue(v reflect.Value, in bool) bool {
	switch v.Kind() {
	case reflect.Ptr:
		return !v.IsNil()
	case reflect.Map, reflect.Slice:
		return v.Len() > 0
	case reflect.String:
		// Non-pointer zero-length strings represent enums and should be kept
		// for Output structs (s3.BucketLocationConstraint is "" for us-east-1)
		return !in || v.Len() > 0
	}
	return true
}
