package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/mxk/awsscan/scan"
	"github.com/mxk/go-cli"
	"github.com/mxk/go-cloud/aws/region"
	"github.com/mxk/go-terraform/tfx"
	"github.com/pkg/errors"

	// Service registration
	_ "github.com/mxk/awsscan/scan/svc"

	// Provider registration
	_ "github.com/mxk/go-terraform/tfaws"
)

type scanCmd struct {
	CA        bool   `flag:"Make CloudAssert-compatible API calls"`
	Hier      string `flag:"Depth or <format> of output hierarchy"`
	Min       bool   `flag:"Minify JSON output"`
	NoRefresh bool   `flag:"Do not refresh Terraform state output"`
	Out       string `flag:"Output <file>"`
	Raw       bool   `flag:"Do not compact output"`
	Regions   string `flag:"Comma-separated <list> of regions (default all)"`
	Roots     bool   `flag:"Make only root API calls"`
	Services  string `flag:"Comma-separated <list> of services (default all)"`
	Stats     bool   `flag:"Report call statistics in output"`
	TFState   bool   `flag:"Generate Terraform state output"`
	Workers   int    `flag:"IPoAC carrier <count>"`
}

func main() {
	cli.Main = cli.Info{
		Usage:   "[options]",
		Summary: "Describe all resources in an AWS account",
		New: func() cli.Cmd {
			return &scanCmd{
				Hier:    "{account}/{region}/{service}.{api},{id}",
				Workers: 64,
			}
		},
	}
	cli.Main.Run(os.Args[1:])
}

func (*scanCmd) Info() *cli.Info { return &cli.Main }

func (*scanCmd) Help(w *cli.Writer) {
	w.Text(`
	Execute List/Get/Describe calls for supported services in one or more
	regions to map resources in an AWS account. The results are written to
	stdout (or file specified by -out) in JSON format:

	  {
	    "<account>/<region>/<service>.<api>": {
	      "<id>": {
	        "src": { "<id>": <out-index>, ... },
	        "in":  { "<param>": <value>, ... },
	        "out": [{ "<field>": <value>, ... }, ...],
	        "err": { <error-info> }
	      }
	    }
	  }

	Use -hier to change the document structure. By default, top-level object
	keys contain the account ID, region, service name, and API name separated by
	'/'. Non-regional services, like IAM, use "aws-global" as their region.
	Service names are the same as AWS SDK package names.

	Second-level objects contain call information indexed by a unique call ID.
	APIs may be called multiple times with different inputs, each one having a
	unique ID. The corresponding call object specifies the inputs, zero or more
	outputs (paginated APIs may produce multiple outputs), and any error
	information returned by AWS.

	Call IDs are base64-encoded SHA-512/256 hashes derived from API inputs,
	name, service, region, and account, making them unique within the document,
	yet stable over multiple scans. Non-root calls contain a "src" field, which
	identifies the outputs (call IDs and output indices) that were used to
	construct the inputs for this call. It is possible to identity all related
	calls for one resource by tracing source call IDs up to the root(s).

	By default, the output is compacted to eliminate noise, such as null values,
	expected errors, and empty objects/arrays. Use -raw option to return an
	exact representation of every API call made during the scan.

	Without -raw, the scan exits with status code 3 if there are any errors in
	the output. With -raw, API errors in the output do not affect exit status.

	Use '-regions help' or '-services help' to see all supported regions or
	services, respectively. With these options, -raw enables JSON output.

	Service names may be negated by using "no-" prefix. For example,
	'-services no-ec2,no-s3' will scan all supported services except ec2 and s3.
	`)
}

func (cmd *scanCmd) Main(args []string) error {
	// Parse and validate command-line options
	keyGen, err := parseHier(cmd.Hier)
	if err != nil {
		return err
	}
	op := scan.Opts{Mode: cmd.mode(), Workers: cmd.Workers}

	// Configure regions and services
	if cmd.Regions != "" {
		if cmd.Regions == "help" {
			return cmd.writeRegions()
		}
		op.Regions = strings.Split(cmd.Regions, ",")
	}
	if cmd.Services != "" {
		if cmd.Services == "help" {
			return cmd.writeServices()
		}
		op.Services = getServices(cmd.Services)
	}

	// Execute scan
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return errors.Wrap(err, "failed to load AWS config")
	}
	maps, err := scan.Account(&cfg, op)
	if err != nil {
		return err
	}

	// Write Terraform state only if no other JSON-related flags are set
	if cmd.TFState && !cmd.Min && !cmd.Raw && !cmd.Stats {
		s, err := scan.NewTFState(maps)
		if err != nil {
			return err
		}
		if !cmd.NoRefresh {
			if s, err = tfx.Context().Refresh(s); err != nil {
				return err
			}
			tfx.Deps.Infer(s)
		}
		return tfx.WriteStateFile(cmd.Out, s)
	}

	// Format and write JSON output
	if makeValues(maps); !cmd.Raw {
		maps = scan.Compact(maps)
	}
	h := makeHier(maps, keyGen, cmd.Stats)
	if err = cmd.writeJSON(h); err == nil && !cmd.Raw {
		if err = apiErr(maps); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			cli.Exit(3)
		}
	}
	return err
}

// mode converts command line options into scan.Mode.
func (cmd *scanCmd) mode() (m scan.Mode) {
	if cmd.CA {
		m |= scan.CloudAssert
	}
	if cmd.Roots {
		m |= scan.RootsOnly
	}
	if cmd.Stats {
		m |= scan.KeepStats
	}
	if cmd.TFState {
		tfx.SetLogFilter(os.Stderr, "", true)
		m |= scan.TFState
	}
	return
}

// writeRegions writes regions within each partition to cmd.Out.
func (cmd *scanCmd) writeRegions() error {
	parts := endpoints.DefaultPartitions()
	regions := make(map[string][]string)
	for _, p := range parts {
		regions[p.ID()] = region.Related(p.ID())
	}
	if cmd.Raw {
		return cmd.writeJSON(regions)
	}
	var b bytes.Buffer
	for _, p := range parts {
		b.WriteString(p.ID())
		b.WriteString(":\n")
		for _, r := range regions[p.ID()] {
			b.WriteString("- ")
			b.WriteString(r)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	return cli.WriteFile(cmd.Out, func(w io.Writer) error {
		_, err := b.WriteTo(w)
		return err
	})
}

// writeServices writes APIs and regions for all supported services to cmd.Out.
func (cmd *scanCmd) writeServices() error {
	var regions []string
	for _, p := range endpoints.DefaultPartitions() {
		regions = append(regions, region.Related(p.ID())...)
	}
	sort.Strings(regions)
	type svc struct {
		API      map[string][]string `json:"api"`
		Regions  []string            `json:"regions"`
		maxWidth int
	}
	maxLen := 0
	svcNames := scan.ServiceNames()
	all := make(map[string]svc, len(svcNames))
	for _, name := range svcNames {
		s := svc{
			Regions: make([]string, 0, len(regions)),
			API:     scan.API(name),
		}
		endpoint := scan.ServiceInfo(name).ID
		for _, r := range regions {
			if region.Supports(r, endpoint) {
				s.Regions = append(s.Regions, r)
			}
		}
		if len(s.API) > maxLen {
			maxLen = len(s.API)
		}
		for name, deps := range s.API {
			if len(deps) > 0 && len(name) > s.maxWidth {
				s.maxWidth = len(name)
			}
		}
		all[name] = s
	}
	if cmd.Raw {
		return cmd.writeJSON(all)
	}
	var b bytes.Buffer
	apiNames := make([]string, 0, maxLen)
	for _, name := range svcNames {
		s := all[name]
		b.WriteString(name)
		b.WriteString(":\n- regions:\n")
		for _, r := range s.Regions {
			b.WriteString("  - ")
			b.WriteString(r)
			b.WriteByte('\n')
		}
		b.WriteString("- api:\n")
		apiNames = apiNames[:0]
		for api := range s.API {
			apiNames = append(apiNames, api)
		}
		sort.Strings(apiNames)
		for _, name := range apiNames {
			b.WriteString("  - ")
			if deps := s.API[name]; len(deps) == 0 {
				b.WriteString(name)
			} else {
				fmt.Fprintf(&b, "%-*s <- ", s.maxWidth, name)
				for i, dep := range s.API[name] {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString(dep)
				}
			}
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	return cli.WriteFile(cmd.Out, func(w io.Writer) error {
		_, err := b.WriteTo(w)
		return err
	})
}

// writeJSON writes the JSON encoding of v to cmd.Out.
func (cmd *scanCmd) writeJSON(v interface{}) error {
	return cli.WriteFile(cmd.Out, func(w io.Writer) error {
		enc := json.NewEncoder(w)
		if enc.SetEscapeHTML(false); !cmd.Min {
			enc.SetIndent("", "\t")
		}
		return errors.Wrap(enc.Encode(v), "failed to encode JSON")
	})
}

// keyGenFunc returns hierarchy keys for the given call.
type keyGenFunc func(m *scan.Map, api string, c *scan.Call) []string

// parseHier returns the hierarchy key generator function for the given spec.
func parseHier(spec string) (keyGenFunc, error) {
	if depth, err := strconv.Atoi(spec); err == nil {
		keys := []string{"{account}", "{region}", "{service}", "{api}", "{id}"}
		if depth <= 0 {
			spec = strings.Join(keys, "/")
		} else if depth >= len(keys)-1 {
			spec = strings.Join(keys, ",")
		} else {
			i := len(keys) - depth
			keys[0] = strings.Join(keys[:i], "/")
			spec = strings.Join(append(keys[:1], keys[i:]...), ",")
		}
	} else if !strings.Contains(spec, "{id}") {
		return nil, errors.New(`hierarchy spec must contain "{id}"`)
	}
	return func(m *scan.Map, api string, c *scan.Call) []string {
		keys := strings.NewReplacer(
			"{account}", m.Account,
			"{region}", m.Region,
			"{service}", m.Service,
			"{api}", api,
			"{id}", c.ID,
		).Replace(spec)
		return strings.Split(keys, ",")
	}, nil
}

// getServices extracts service names from the spec.
func getServices(spec string) []string {
	if !strings.Contains(spec, "no-") {
		return strings.Split(spec, ",")
	}
	include := make(map[string]bool)
	exclude := make(map[string]bool)
	for _, name := range strings.Split(spec, ",") {
		if ex := strings.TrimPrefix(name, "no-"); len(ex) == len(name) {
			include[name] = true
			delete(exclude, name)
		} else {
			exclude[ex] = true
			delete(include, ex)
		}
	}
	all := scan.ServiceNames()
	keep := all[:0]
	for _, name := range all {
		if !exclude[name] && (len(include) == 0 || include[name]) {
			keep = append(keep, name)
		}
	}
	return keep
}

// makeValues replaces nil input/output values with empty equivalents to produce
// correct JSON output (e.g. nil slice -> null, empty slice -> []).
func makeValues(maps []*scan.Map) {
	shouldVisit := func(t reflect.Type) bool {
		for {
			switch t.Kind() {
			case reflect.Ptr:
				t = t.Elem()
			case reflect.Map, reflect.Slice, reflect.Struct:
				return true
			default:
				return false
			}
		}
	}
	emptyVals := make(map[reflect.Type]reflect.Value)
	makeEmpty := func(t reflect.Type) reflect.Value {
		v, ok := emptyVals[t]
		if !ok {
			if t.Kind() == reflect.Slice {
				v = reflect.MakeSlice(t, 0, 0)
			} else {
				v = reflect.MakeMapWithSize(t, 0)
			}
			emptyVals[t] = v
		}
		return v
	}
	var visit func(v reflect.Value)
	visit = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Ptr:
			if !v.IsNil() {
				visit(v.Elem())
			}
		case reflect.Map:
			if v.IsNil() {
				v.Set(makeEmpty(v.Type()))
			} else if t := v.Type().Elem(); shouldVisit(t) {
				tmp := reflect.New(t).Elem()
				for _, k := range v.MapKeys() {
					tmp.Set(v.MapIndex(k))
					visit(tmp)
					v.SetMapIndex(k, tmp)
				}
			}
		case reflect.Slice:
			if v.IsNil() {
				v.Set(makeEmpty(v.Type()))
			} else if shouldVisit(v.Type().Elem()) {
				for i := v.Len() - 1; i >= 0; i-- {
					visit(v.Index(i))
				}
			}
		case reflect.Struct:
			for i := v.NumField() - 1; i >= 0; i-- {
				if f := v.Field(i); f.CanSet() {
					visit(f)
				}
			}
		}
	}
	scan.Walk(maps, func(_ *scan.Map, _ string, c *scan.Call) error {
		for _, out := range c.Out {
			visit(reflect.ValueOf(out))
		}
		visit(reflect.ValueOf(c.In))
		return nil
	})
}

// hier is one level in the scan result hierarchy.
type hier map[string]interface{}

// makeHier arranges scan results into a hierarchy.
func makeHier(maps []*scan.Map, keyGen keyGenFunc, stats bool) hier {
	root := make(hier)
	scan.Walk(maps, func(m *scan.Map, api string, c *scan.Call) error {
		h, keys := root, keyGen(m, api, c)
		for _, k := range keys[:len(keys)-1] {
			next, _ := h[k].(hier)
			if next == nil {
				next = make(hier)
				h[k] = next
			}
			h = next
		}
		h[keys[len(keys)-1]] = c
		return nil
	})
	if stats {
		const key = "#stats"
		var addStats func(h hier) *scan.Stats
		addStats = func(h hier) *scan.Stats {
			s := new(scan.Stats)
			for _, v := range h {
				switch v.(type) {
				case hier:
					for _, v := range h {
						t := addStats(v.(hier))
						s.Combine(t)
						t.RoundTimes()
					}
				case *scan.Call:
					for _, v := range h {
						t := v.(*scan.Call).Stats
						s.Combine(t)
						t.RoundTimes()
					}
				}
				break
			}
			h[key] = s
			return s
		}
		addStats(root)
		root[key].(*scan.Stats).RoundTimes()
	}
	return root
}

// apiErr returns the first API error in m.
func apiErr(maps []*scan.Map) error {
	return scan.Walk(maps, func(m *scan.Map, api string, c *scan.Call) error {
		if c.Err != nil {
			return errors.Errorf("%s/%s/%s.%s: %v",
				m.Account, m.Region, m.Service, api, c.Err)
		}
		return nil
	})
}
