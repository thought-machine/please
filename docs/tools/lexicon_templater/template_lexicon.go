package main

import (
	"encoding/json"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/peterebden/go-cli-init/v5/flags"
)

type rules struct {
	Functions map[string]*rule `json:"functions"`
}

// Named returns the rule with given name.
func (r *rules) Named(name string) *rule {
	rule, present := r.Functions[name]
	if !present {
		panic("unknown function " + name)
	}
	rule.Name = name // Not actually stored in the JSON, but useful in the template.
	return rule
}

// AddLinks adds HTML links to a function docstring.
func (r *rules) AddLinks(name, docstring string) string {
	if strings.Contains(name, "_") { // Don't do it for something generic like "tarball"
		for k := range r.Functions {
			var re = regexp.MustCompile("\b(" + k + ")\b")
			if name == k {
				docstring = re.ReplaceAllString(docstring, "<code>$1</code>")
			} else {
				docstring = re.ReplaceAllString(docstring, `<a href="#$1">$1</a>`)
			}
		}
	}
	return docstring
}

type rule struct {
	Args []struct {
		Comment    string   `json:"comment"`
		Deprecated bool     `json:"deprecated"`
		Name       string   `json:"name"`
		Required   bool     `json:"required"`
		Types      []string `json:"types"`
	} `json:"args"`
	Aliases   []string `json:"aliases"`
	Docstring string   `json:"docstring"`
	Comment   string   `json:"comment"`
	Language  string   `json:"language"`
	Name      string
}

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
	r := &rules{}
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
