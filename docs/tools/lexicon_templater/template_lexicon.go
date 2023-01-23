package main

import (
	"encoding/json"
	htmltemplate "html/template"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/peterebden/go-cli-init/v5/flags"
)

type rules struct {
	Functions map[string]*rule `json:"functions"`
}

// // PluginRules returns the rules for this plugin.
// func (r *rules) PluginRules(plugin string) []*rule {
// 	var rules []*rule
// 	for _, rule := range r.Functions {
// 		if rule.Plugin == plugin {
// 			rules = append(rules, rule)
// 		}
// 	}
// 	return rules
// }

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
	Input []string `short:"i" long:"input" description:"Input file(s)"`
	Rules []string `short:"r" long:"rules" description:"Rules file"`
}

func main() {
	flags.ParseFlagsOrDie("Docs template", &opts, nil)
	r := &rules{}
	split := strings.Split(opts.Input[0], "/")
	basename := split[len(split)-1]
	tmpl, err := template.New(basename).Funcs(template.FuncMap{
		"join": strings.Join,
		"newlines": func(name, docstring string) string {
			return r.AddLinks(name, strings.ReplaceAll(htmltemplate.HTMLEscapeString(docstring), "\n", "<br/>"))
		},
	}).ParseFiles(opts.Input...)
	must(err)
	for _, rulesRile := range opts.Rules {
		b, err := os.ReadFile(rulesRile)
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
