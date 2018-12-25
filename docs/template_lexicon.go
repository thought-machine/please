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
	Functions map[string]rule `json:"functions"`
}

// Named returns the rule with given name.
func (r *rules) Named(name string) rule {
	rule, present := r.Functions[name]
	if !present {
		panic("unknown function " + name)
	}
	rule.Name = name // Not actually stored in the JSON, but useful in the template.
	return rule
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
	tmpl, err := template.New("lexicon.html").Funcs(template.FuncMap{
		"join": strings.Join,
		"newlines": func(s string) string {
			return strings.Replace(htmltemplate.HTMLEscapeString(s), "\n", "<br/>", -1)
		},
	}).ParseFiles(
		"docs/lexicon.html", "docs/lexicon_entry.html")
	must(err)
	b, err := ioutil.ReadFile("docs/rules.json")
	must(err)
	r := &rules{}
	must(json.Unmarshal(b, r))
	must(tmpl.Execute(os.Stdout, r))
}
