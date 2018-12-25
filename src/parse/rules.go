package parse

import (
	"encoding/json"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/src/parse/rules"
)

// PrintRuleArgs prints the arguments of all builtin rules (plus any associated ones from the given targets)
func PrintRuleArgs(state *core.BuildState, labels []core.BuildLabel) {
	env := getRuleArgs(state, labels)
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		log.Fatalf("Failed JSON encoding: %s", err)
	}
	os.Stdout.Write(b)
}

// AllBuiltinFunctions returns all the builtin functions, including those in a given set of labels.
func AllBuiltinFunctions(state *core.BuildState, labels []core.BuildLabel) map[string]*asp.FuncDef {
	p := newAspParser(state)
	m := map[string]*asp.FuncDef{}
	dir, _ := rules.AssetDir("")
	sort.Strings(dir)
	for _, filename := range dir {
		if !strings.HasSuffix(filename, ".gob") && filename != "builtins.build_defs" {
			stmts, err := p.ParseData(rules.MustAsset(filename), filename)
			if err != nil {
				log.Fatalf("%s", err)
			}
			addAllFunctions(m, stmts)
		}
	}
	for _, l := range labels {
		t := state.Graph.TargetOrDie(l)
		for _, out := range t.Outputs() {
			stmts, err := p.ParseFileOnly(path.Join(t.OutDir(), out))
			if err != nil {
				log.Fatalf("%s", err)
			}
			addAllFunctions(m, stmts)
		}
	}
	return m
}

// addAllFunctions adds all the functions from a set of statements to the given map.
func addAllFunctions(m map[string]*asp.FuncDef, stmts []*asp.Statement) {
	for _, stmt := range stmts {
		if f := stmt.FuncDef; f != nil && !f.IsPrivate && f.Docstring != "" {
			f.Docstring = strings.TrimSpace(strings.Trim(f.Docstring, `"`))
			m[f.Name] = f
		}
	}
}

// getRuleArgs retrieves the arguments of builtin rules. It's split from PrintRuleArgs for testing.
func getRuleArgs(state *core.BuildState, labels []core.BuildLabel) environment {
	argsRegex := regexp.MustCompile("\n +Args: *\n")
	env := environment{Functions: map[string]function{}}
	for name, f := range AllBuiltinFunctions(state, labels) {
		r := function{Docstring: f.Docstring}
		if strings.HasSuffix(f.EoDef.Filename, "_rules.build_defs") {
			r.Language = strings.TrimSuffix(f.EoDef.Filename, "_rules.build_defs")
		}
		if indices := argsRegex.FindStringIndex(r.Docstring); indices != nil {
			r.Comment = strings.TrimSpace(r.Docstring[:indices[0]])
		}
		r.Args = make([]functionArg, len(f.Arguments))
		for i, a := range f.Arguments {
			r.Args[i] = functionArg{
				Name:     a.Name,
				Types:    a.Type,
				Required: a.Value == nil,
			}
			regex := regexp.MustCompile(a.Name + `(?: \(.*\))?: ((?s:.*))`)
			if match := regex.FindStringSubmatch(r.Docstring); match != nil {
				r.Args[i].Comment = filterMatch(match[1])
			}
		}
		env.Functions[name] = r
	}
	return env
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

// filterMatch filters a regex match to the part we want.
// It's pretty much impossible to handle this as just a regex so we do it here instead.
func filterMatch(match string) string {
	lines := strings.Split(match, "\n")
	regex := regexp.MustCompile(`^ *[a-z_]+(?: \([^)]+\))?:`)
	for i, line := range lines {
		if regex.MatchString(line) {
			return strings.Join(lines[:i], " ")
		}
		lines[i] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(match)
}
