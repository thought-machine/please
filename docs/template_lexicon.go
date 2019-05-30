package main

import (
	"encoding/json"
	htmltemplate "html/template"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
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
			if name == k {
				docstring = strings.Replace(docstring, " "+k, " <code>"+k+"</code>", -1)
			} else {
				docstring = strings.Replace(docstring, " "+k, ` <a href="#`+k+`">`+k+"</a>", -1)
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

func main() {
	r := &rules{}
	tmpl, err := template.New("lexicon.html").Funcs(template.FuncMap{
		"join": strings.Join,
		"newlines": func(name, docstring string) string {
			return r.AddLinks(name, strings.Replace(htmltemplate.HTMLEscapeString(docstring), "\n", "<br/>", -1))
		},
	}).ParseFiles("docs/lexicon.html", "docs/lexicon_entry.html")
	must(err)
	b, err := ioutil.ReadFile("docs/rules.json")
	must(err)
	must(json.Unmarshal(b, r))
	for name, rule := range r.Functions {
		if strings.HasPrefix(name, "c_") {
			rule.Aliases = append(rule.Aliases, "c"+name)
		}
	}
	must(tmpl.Execute(os.Stdout, r))
}
