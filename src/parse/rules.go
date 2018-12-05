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

// getRuleArgs retrieves the arguments of builtin rules. It's split from PrintRuleArgs for testing.
func getRuleArgs(state *core.BuildState, labels []core.BuildLabel) environment {
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

func (env *environment) AddAll(stmts []*asp.Statement) {
	argsRegex := regexp.MustCompile("\n +Args: *\n")
	for _, stmt := range stmts {
		if f := stmt.FuncDef; f != nil && f.Name[0] != '_' && f.Docstring != "" {
			r := function{
				Docstring: strings.TrimSpace(strings.Trim(f.Docstring, `"`)),
			}
			if strings.HasSuffix(stmt.Pos.Filename, "_rules.build_defs") {
				r.Language = strings.TrimSuffix(stmt.Pos.Filename, "_rules.build_defs")
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
			env.Functions[f.Name] = r
		}
	}
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
