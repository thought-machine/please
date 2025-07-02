package main

import (
	"encoding/json"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/peterebden/go-cli-init/v5/flags"

	"github.com/thought-machine/please/docs/tools/lexicon_templater/rules"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var opts struct {
	Input []string `short:"i" long:"input" required:"true" description:"Input file(s)"`
	Args  struct {
		Rules []string `positional-arg-name:"Rules files" required:"true" description:"Rules file(s)"`
	} `positional-args:"true"`
}

func main() {
	flags.ParseFlagsOrDie("Docs template", &opts, nil)
	r := &rules.Rules{}
	basename := filepath.Base(opts.Input[0])
	tmpl, err := template.New(basename).Funcs(template.FuncMap{
		"join": strings.Join,
		"newlines": func(name, docstring string) string {
			return r.AddLinks(name, strings.ReplaceAll(htmltemplate.HTMLEscapeString(docstring), "\n", "<br/>"))
		},
	}).ParseFiles(opts.Input...)
	must(err)
	for _, rulesFile := range opts.Args.Rules {
		b, err := os.ReadFile(rulesFile)
		must(err)
		must(json.Unmarshal(b, r))
	}
	for name, rule := range r.Functions {
		if strings.HasPrefix(name, "c_") {
			rule.Aliases = append(rule.Aliases, "c"+name)
		}
	}
	must(tmpl.Execute(os.Stdout, r))
}
