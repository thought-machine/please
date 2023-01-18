package help

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
)

var log = logging.Log

// PrintRuleArgs prints the arguments of all builtin rules
func PrintRuleArgs() {
	env := getRuleArgs(newState())
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		log.Fatalf("Failed JSON encoding: %s", err)
	}
	os.Stdout.Write(b)
}

func newState() *core.BuildState {
	// If we're in a repo, we might be able to read some stuff from there.
	if core.FindRepoRoot() {
		if config, err := core.ReadDefaultConfigFiles(nil); err == nil {
			return core.NewBuildState(config)
		}
	}
	return core.NewDefaultBuildState()
}

// AllBuiltinFunctions returns all the builtin functions, including any defined
// by the config (e.g. PreloadBuildDefs or BuildDefsDir).
// You are guaranteed that every statement in the returned map is a FuncDef.
func AllBuiltinFunctions(state *core.BuildState) map[string]*asp.Statement {
	p := asp.NewParser(state)
	m := map[string]*asp.Statement{}
	dir, _ := rules.AllAssets(state.ExcludedBuiltinRules())
	sort.Strings(dir)
	for _, filename := range dir {
		if filename != "builtins.build_defs" {
			assetSrc, err := rules.ReadAsset(filename)
			if err != nil {
				log.Fatalf("Failed to read an asset %s", filename)
			}
			if stmts, err := p.ParseData(assetSrc, filename); err == nil {
				addAllFunctions(m, stmts, true)
			}
		}
	}
	for _, preload := range state.Config.Parse.PreloadBuildDefs {
		if stmts, err := p.ParseFileOnly(preload); err != nil {
			addAllFunctions(m, stmts, false)
		}
	}
	for _, dir := range state.Config.Parse.BuildDefsDir {
		if files, err := os.ReadDir(dir); err == nil {
			for _, file := range files {
				if !file.IsDir() {
					if stmts, err := p.ParseFileOnly(filepath.Join(dir, file.Name())); err == nil {
						addAllFunctions(m, stmts, false)
					}
				}
			}
		}
	}
	return m
}

// addAllFunctions adds all the functions from a set of statements to the given map.
func addAllFunctions(m map[string]*asp.Statement, stmts []*asp.Statement, builtin bool) {
	for _, stmt := range stmts {
		if f := stmt.FuncDef; f != nil && !f.IsPrivate && f.Docstring != "" {
			f.Docstring = strings.TrimSpace(strings.Trim(f.Docstring, `"`))
			f.IsBuiltin = builtin
			args := make([]asp.Argument, 0, len(f.Arguments))
			for _, arg := range f.Arguments {
				if !arg.IsPrivate {
					args = append(args, arg)
				}
			}
			f.Arguments = args
			m[f.Name] = stmt
		}
	}
}

// getRuleArgs retrieves the arguments of builtin rules. It's split from PrintRuleArgs for testing.
func getRuleArgs(state *core.BuildState) environment {
	argsRegex := regexp.MustCompile("\n +Args: *\n")
	env := environment{Functions: map[string]function{}}
	for name, stmt := range AllBuiltinFunctions(state) {
		f := stmt.FuncDef
		r := function{Docstring: f.Docstring}
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
}

// A functionArg represents a single argument to a function.
type functionArg struct {
	Comment    string   `json:"comment,omitempty"`
	Name       string   `json:"name"`
	Deprecated bool     `json:"deprecated,omitempty"`
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
