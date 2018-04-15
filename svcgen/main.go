// +build codegen

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/private/model/api"
)

const usage = `
Usage: %s [options] <out-dir> <service> [<service> ...]

Generates scanner template code for each AWS service and writes the output to
out-dir. If out-dir is "-", the output is written to stdout. Services can be
specified by package name (e.g. "iam") or as paths to existing files, in which
case the service name is extracted from the file name.

The generated template must be edited to remove unnecessary root API calls and
to fill-in the required input parameters of non-root calls. This tool was
designed to speed-up the process of adding a new scanner, not to generate all
required code automatically.

Options:
`

var (
	modelsDir = filepath.Join(findSDK(), "models")
	scanAPI   = []string{"Describe", "Get", "List"}
)

func init() {
	flag.Usage = func() {
		bin := filepath.Base(os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), usage[1:], bin)
		flag.PrintDefaults()
	}
	flag.StringVar(&modelsDir, "models", modelsDir, "SDK models `dir`ectory")
	if flag.Parse(); flag.NArg() < 2 {
		flag.Usage()
		os.Exit(2)
	}
	cwd, err := os.Getwd()
	must(err)
	must(os.Chdir(modelsDir))
	must(api.Bootstrap())
	must(os.Chdir(cwd))
}

func main() {
	outDir := flag.Arg(0)
	if outDir != "-" {
		must(os.MkdirAll(outDir, 0755))
	}
	rc := 0
	names := svcNames(flag.Args()[1:])
	svcs := loadSvcs(names)
	for _, name := range names {
		fmt.Fprintln(os.Stderr, "==>", name)
		if svc := svcs[name]; svc == nil {
			fmt.Fprintln(os.Stderr, "Service not found")
		} else if buf, err := svc.render(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if outDir == "-" {
			os.Stdout.Write(buf)
			continue
		} else {
			dst := filepath.Join(outDir, name+".go")
			if _, err = os.Stat(dst); !os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "File already exists:", dst)
			} else if err = ioutil.WriteFile(dst, buf, 0644); err != nil {
				fmt.Fprintln(os.Stderr, err)
			} else {
				continue
			}
		}
		rc = 1
	}
	os.Exit(rc)
}

// svcNames extracts service names from the spec, which may contain file names.
func svcNames(spec []string) (names []string) {
	var err error
	for _, s := range spec {
		v := []string{s}
		if strings.ContainsAny(s, "*?[") {
			v, err = filepath.Glob(s)
			must(err)
		}
		for _, name := range v {
			name = filepath.Base(name)
			if i := strings.IndexByte(name, '.'); i != -1 {
				name = name[:i]
			}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return
}

// Svc describes the API for one AWS service. It is passed in to svcTpl for
// rendering.
type Svc struct {
	api.API
	Name string
}

// svcTpl is the template used to render scanner code for each service.
var svcTpl = template.Must(template.New("").Funcs(template.FuncMap{
	"join": func(s []string) string { return strings.Join(s, ", ") },
}).Parse(`
package service

import (
	"github.com/LuminalHQ/cloudcover/awsscan/scan"
	"github.com/aws/aws-sdk-go-v2/service/{{.Name}}"
)

type {{.Type}} struct{}

func init() { scan.Register({{.Type}}{}) }

func ({{.Type}}) Name() string         { return {{.Name}}.ServiceName }
func ({{.Type}}) NewFunc() interface{} { return {{.Name}}.New }
func ({{.Type}}) Roots() []interface{} {
	return []interface{}{
	{{- range .Roots}}
		[]{{.}}{nil},
	{{- end}}
	}
}

{{- range .NonRoots}}

func ({{$.Type}}) {{.ExportedName}}() (q []{{.InputRef.Shape.GoTypeWithPkgName}}) {
	// Required: {{join .InputRef.Shape.Required}}
	return
}
{{- end}}
`[1:]))

// loadSvcs loads API definitions for the specified service names.
func loadSvcs(names []string) map[string]*Svc {
	m := make(map[string]*Svc, len(names))
	for _, name := range names {
		m[name] = nil
	}
	// All api-2.json files must be parsed because package names don't match
	// directory names and newer versions should replace older ones.
	root := filepath.Join(modelsDir, "apis")
	walk := func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.Name() != "api-2.json" {
			return err
		}
		var s Svc
		s.Attach(path)
		s.Name = s.PackageName()
		if _, want := m[s.Name]; want {
			s.Setup()
			m[s.Name] = &s
		}
		return nil
	}
	if err := filepath.Walk(root, walk); err != nil {
		panic(err)
	}
	return m
}

// Type returns the service scanner type name.
func (s *Svc) Type() string { return s.Name + "Svc" }

// Roots returns input types for scanner API calls that have no required
// parameters.
func (s *Svc) Roots() []string {
	var types []string
	for _, op := range s.Operations {
		if isScanAPI(op, true) {
			types = append(types, op.InputRef.Shape.GoTypeWithPkgName())
		}
	}
	sort.Strings(types)
	return types
}

// NonRoots returns all scanner API calls that have at least one required
// parameter.
func (s *Svc) NonRoots() []*api.Operation {
	var ops []*api.Operation
	for _, op := range s.Operations {
		if isScanAPI(op, false) {
			ops = append(ops, op)
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].ExportedName < ops[j].ExportedName
	})
	return ops
}

// render generates scanner code for the specified service.
func (s *Svc) render() ([]byte, error) {
	tpl, err := svcTpl.Clone()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(make([]byte, 0, 8192))
	if err = tpl.Execute(buf, s); err != nil {
		return nil, err
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		err = fmt.Errorf("Source:\n%s\nError: %v\n", buf.Bytes(), err)
	}
	return src, err
}

// findSDK returns the absolute path to the aws-sdk-go-v2 directory used for
// compiling the binary.
func findSDK() string {
	fn := runtime.FuncForPC(reflect.ValueOf(api.Bootstrap).Pointer())
	path, _ := fn.FileLine(fn.Entry())
	if !filepath.IsAbs(path) {
		panic("svcgen: invalid path to api.go: " + path)
	}
	for filepath.Base(path) != "aws-sdk-go-v2" {
		prev := path
		if path = filepath.Dir(path); path == prev {
			panic("svcgen: aws-sdk-go-v2 directory not found")
		}
	}
	return path
}

// isScanAPI returns true if op is an API call that may be used by the scanner.
func isScanAPI(op *api.Operation, isRoot bool) bool {
	if !op.Deprecated && isRoot == (len(op.InputRef.Shape.Required) == 0) {
		for _, prefix := range scanAPI {
			if strings.HasPrefix(op.ExportedName, prefix) {
				return true
			}
		}
	}
	return false
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
