package rules

import (
	"regexp"
	"strings"
)

type Rules struct {
	Functions map[string]*Rule `json:"functions"`
}

// Named returns the Rule with given name.
func (r *Rules) Named(name string) *Rule {
	rule, present := r.Functions[name]
	if !present {
		panic("unknown function " + name)
	}
	rule.Name = name // Not actually stored in the JSON, but useful in the template.
	return rule
}

// AddLinks adds HTML links to a function docstring.
func (r *Rules) AddLinks(name, docstring string) string {
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

type Rule struct {
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
