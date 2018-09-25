// Package main implements a build provider for Please that understands Go files.
// This could be considered a base for such a thing; it is not complete in regard to
// all the subtleties of how Go would process them, and misses a lot of features
// (like useful cross-package dependencies, for example).
package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strings"
	"text/template"

	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("go_provider")

type Request struct {
	Rule string `json:"rule"`
}

type Response struct {
	Rule      string   `json:"rule"`
	Success   bool     `json:"success"`
	Messages  []string `json:"messages"`
	BuildFile string   `json:"build_file"`
}

var tmpl = template.Must(template.New("build").Funcs(template.FuncMap{
	"filter": func(in map[string]*ast.File, test bool) []string {
		ret := []string{}
		for name := range in {
			if strings.HasSuffix(name, "test.go") == test {
				ret = append(ret, path.Base(name))
			}
		}
		return ret
	},
}).Parse(`
{{ range $pkgName, $pkg := . }}
go_library(
    name = "{{ $pkgName }}",
    srcs = [
        {{ range filter $pkg.Files false }}
        "{{ . }}",
        {{ end }}
    ],
)

{{ if filter $pkg.Files true }}
go_test(
    name = "{{ $pkgName }}_test",
    srcs = [
        {{ range filter $pkg.Files true }}
        "{{ . }}",
        {{ end }}
    ],
    deps = [":{{ $pkgName }}"],
)
{{ end }}
{{ end }}
`))

func provide(ch chan<- *Response, dir string) {
	contents, err := parse(dir)
	resp := &Response{
		Rule:      dir,
		BuildFile: contents,
	}
	if err != nil {
		resp.Messages = []string{err.Error()}
	}
	ch <- resp
}

func parse(dir string) (string, error) {
	var b strings.Builder
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, dir, nil, parser.ImportsOnly)
	if err != nil {
		return "", err
	} else if err := tmpl.Execute(&b, pkgs); err != nil {
		return "", err
	}
	return b.String(), nil
}

func main() {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	ch := make(chan *Response, 10)
	go func() {
		for resp := range ch {
			if err := encoder.Encode(resp); err != nil {
				log.Error("Failed to encode message: %s", err)
			}
		}
	}()
	for {
		req := &Request{}
		if err := decoder.Decode(req); err != nil {
			log.Error("Failed to decode incoming message: %s", err)
			continue
		}
		go provide(ch, req.Rule)
	}
}
