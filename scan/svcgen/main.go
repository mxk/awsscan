// +build codegen

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/mxk/cloudcover/awsscan/scan"
	"github.com/mxk/cloudcover/x/cli"
	"github.com/mxk/cloudcover/x/gomod"
	"github.com/aws/aws-sdk-go-v2/private/model/api"
)

var genCli = cli.Info{
	Usage:   "[options] <out-dir> <service> [<service> ...]",
	MinArgs: 2,
	New: func() cli.Cmd {
		return &genCmd{
			Models: filepath.Join(gomod.Root(api.Bootstrap).Path(), "models"),
			API:    "BatchGet,Describe,Get,List",
		}
	},
}

var scanAPI []string

func main() { genCli.Run(os.Args[1:]) }

type genCmd struct {
	Models string `flag:"SDK models <directory>"`
	API    string `flag:"Comma-separated API <prefixes>"`
}

func (*genCmd) Info() *cli.Info { return &genCli }

func (*genCmd) Help(w *cli.Writer) {
	w.Text(`
	Generates scanner template code for each AWS service and writes the output
	to out-dir. If out-dir is "-", the output is written to stdout. Services can
	be specified by package name (e.g. "iam") or as paths to existing files, in
	which case the service name is extracted from the file name.

	The generated template must be edited to remove unnecessary root API calls
	and to fill-in the required input parameters of non-root calls. This tool is
	designed to speed-up the process of adding a new scanner, not to generate
	all required code automatically.
	`)
}

func (cmd *genCmd) Main(args []string) error {
	cwd, err := os.Getwd()
	must(err)
	must(os.Chdir(cmd.Models))
	must(api.Bootstrap())
	must(os.Chdir(cwd))
	outDir := args[0]
	if outDir != "-" {
		must(os.MkdirAll(outDir, 0755))
	}
	scanAPI = strings.Split(cmd.API, ",")
	rc := 0
	names := svcNames(args[1:])
	svcs := loadSvcs(cmd.Models, names)
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
	cli.Exit(rc)
	return nil
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
	Name    string
	ScanPkg string
}

// svcTpl is the template used to render scanner code for each service.
var svcTpl = template.Must(template.New("").Funcs(template.FuncMap{
	"join": func(s []string) string { return strings.Join(s, ", ") },
}).Parse(`
package svc

import (
	"{{.ScanPkg}}"
	"github.com/aws/aws-sdk-go-v2/service/{{.Name}}"
)

type {{.Type}} struct{ *scan.Ctx }

var _ = scan.Register({{.Name}}.EndpointsID, {{.Name}}.New, {{.Type}}{},
	{{- range .Roots}}
		[]{{.}}{},
	{{- end}}
)

{{- range .NonRoots}}

func (s {{$.Type}}) {{.ExportedName}}() (q []{{.InputRef.Shape.GoTypeWithPkgNameElem}}) {
	// Required: {{join .InputRef.Shape.Required}}
	return
}
{{- end}}
`[1:]))

// loadSvcs loads API definitions for the specified service names.
func loadSvcs(modelsDir string, names []string) map[string]*Svc {
	m := make(map[string]*Svc, len(names))
	for _, name := range names {
		m[name] = nil
	}
	scanPkg := reflect.TypeOf(scan.Map{}).PkgPath()
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
		s.ScanPkg = scanPkg
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
			types = append(types, op.InputRef.Shape.GoTypeWithPkgNameElem())
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
