package parse

import (
	"encoding/json"
	"os"
	"path"
	"sort"
	"strings"

	"core"
	"parse/asp"
	"parse/rules"
)

// PrintRuleArgs prints the arguments of all builtin rules (plus any associated ones from the given targets)
func PrintRuleArgs(state *core.BuildState, labels []core.BuildLabel) {
	p := newAspParser(state)
	env := environment{Functions: map[string]function{}}
	dir, _ := rules.AssetDir("")
	sort.Strings(dir)
	for _, filename := range dir {
		if !strings.HasSuffix(filename, ".gob") && filename != "builtins.build_defs" {
			stmts, err := p.ParseData(rules.MustAsset(filename), filename)
			if err != nil {
				log.Fatalf("%s", err)
			}
			env.AddAll(stmts)
		}
	}
	for _, l := range labels {
		t := state.Graph.TargetOrDie(l)
		for _, out := range t.Outputs() {
			stmts, err := p.ParseFileOnly(path.Join(t.OutDir(), out))
			if err != nil {
				log.Fatalf("%s", err)
			}
			env.AddAll(stmts)
		}
	}
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		log.Fatalf("Failed JSON encoding: %s", err)
	}
	os.Stdout.Write(b)
}

type environment struct {
	Functions map[string]function `json:"functions"`
}

// A function describes a function within the global environment
type function struct {
	Args      []functionArg `json:"args"`
	Comment   string        `json:"comment,omitempty"`
	Docstring string        `json:"docstring,omitempty"`
	Language  string        `json:"language,omitempty"`
}

// A functionArg represents a single argument to a function.
type functionArg struct {
	Comment    string   `json:"comment,omitempty"`
	Deprecated bool     `json:"deprecated,omitempty"`
	Name       string   `json:"name"`
	Required   bool     `json:"required,omitempty"`
	Types      []string `json:"types"`
}

func (env *environment) AddAll(stmts []*asp.Statement) {
	for _, stmt := range stmts {
		if f := stmt.FuncDef; f != nil && f.Name[0] != '_' {
			r := function{
				Docstring: f.Docstring,
				Comment:   f.Docstring,
			}
			if idx := strings.IndexRune(f.Docstring, '\n'); idx != -1 {
				r.Comment = f.Docstring[:idx]
			}
			r.Args = make([]functionArg, len(f.Arguments))
			for i, a := range f.Arguments {
				r.Args[i] = functionArg{
					Name:     a.Name,
					Types:    a.Type,
					Required: a.Value == nil,
				}
			}
			env.Functions[f.Name] = r
		}
	}
}
