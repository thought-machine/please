package langserver

import (
	"core"
	"parse/asp"
	"parse/rules"
	"path"
	"testing"
	"tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

func TestNewAnalyzer(t *testing.T) {
	a := newAnalyzer()
	assert.NotEqual(t, nil, a.BuiltIns)

	goLibrary := a.BuiltIns["go_library"]
	assert.Equal(t, 15, len(goLibrary.ArgMap))
	assert.Equal(t, true, goLibrary.ArgMap["name"].required)
}

func TestIdentFromPos(t *testing.T) {
	a := newAnalyzer()
	core.FindRepoRoot()

	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/test_data/example.build")
	uri := lsp.DocumentURI("file://" + filepath)

	ident, _ := a.IdentFromPos(uri, lsp.Position{Line: 8, Character: 5})
	assert.NotEqual(t, nil, ident)
	assert.Equal(t, "go_library", ident.Name)

	ident, _ = a.IdentFromPos(uri, lsp.Position{Line: 18, Character: 0})
	assert.True(t, nil == ident)
}

func TestIdentFromFile(t *testing.T) {
	a := newAnalyzer()
	core.FindRepoRoot()

	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/test_data/example.build")
	uri := lsp.DocumentURI("file://" + filepath)

	idents, err := a.IdentFromFile(uri)
	assert.Equal(t, err, nil)
	assert.Equal(t, idents[0].Name, "go_library")
	assert.Equal(t, idents[0].StartLine, 0)
	assert.Equal(t, idents[0].EndLine, 17)

	assert.Equal(t, idents[1].Name, "go_test")
	assert.Equal(t, idents[1].StartLine, 19)
	assert.Equal(t, idents[1].EndLine, 31)
}

func TestNewRuleDef(t *testing.T) {
	a := newAnalyzer()

	// Test header the definition for build_rule
	expected := "def go_library(name:str, srcs:list, asm_srcs:list=None, hdrs:list=None, out:str=None, deps:list=[],\n" +
		"               visibility:list=None, test_only:bool&testonly=False, complete:bool=True,\n" +
		"               _needs_transitive_deps=False, _all_srcs=False, cover:bool=True,\n" +
		"               filter_srcs:bool=True, _link_private:bool=False, _link_extra:bool=True)"

	ruleContent := rules.MustAsset("go_rules.build_defs")

	statements, err := a.parser.ParseData(ruleContent, "go_rules.build_defs")
	assert.Equal(t, err, nil)

	stmt := getStatementByName(statements, "go_library")
	ruleDef := newRuleDef(string(ruleContent), stmt)
	//t.Log(getStatementByName(statements, "cgo_library").EndPos)
	assert.Equal(t, ruleDef.Header, expected)
	assert.Equal(t, len(ruleDef.Arguments), len(ruleDef.ArgMap))
	assert.Equal(t, false, ruleDef.ArgMap["_link_private"].required)
	assert.Equal(t, true, ruleDef.ArgMap["name"].required)
	assert.Equal(t, ruleDef.ArgMap["visibility"].definition,
		"visibility required:false, type:list")

	// Test header for len()
	ruleContent = rules.MustAsset("builtins.build_defs")

	statements, err = a.parser.ParseData(ruleContent, "builtins.build_defs")
	assert.Equal(t, err, nil)

	stmt = getStatementByName(statements, "len")
	ruleDef = newRuleDef(string(ruleContent), stmt)

	assert.Equal(t, ruleDef.Header, "def len(obj)")
	assert.Equal(t, len(ruleDef.ArgMap), 1)
	assert.Equal(t, true, ruleDef.ArgMap["obj"].required)
	assert.Equal(t, ruleDef.ArgMap["obj"].definition, "obj required:true")

	// Test header for a string function, startswith()
	stmt = getStatementByName(statements, "startswith")
	ruleDef = newRuleDef(string(ruleContent), stmt)

	assert.Equal(t, ruleDef.Header, "str.startswith(s:str)")
	assert.Equal(t, len(ruleDef.ArgMap), 1)
	assert.Equal(t, true, ruleDef.ArgMap["s"].required)

	// Test header for a config function, setdefault()
	stmt = getStatementByName(statements, "setdefault")
	ruleDef = newRuleDef(string(ruleContent), stmt)

	assert.Equal(t, ruleDef.Header, "config.setdefault(key:str, default=None)")
	assert.Equal(t, 2, len(ruleDef.ArgMap))
	assert.Equal(t, false, ruleDef.ArgMap["default"].required)
}

func TestGetArgString(t *testing.T) {
	argWithVal := asp.Argument{
		Name: "mystring",
		Type: []string{"string", "list"},
		Value: &asp.Expression{
			Optimised: &asp.OptimisedExpression{
				Local: "None",
			},
		},
	}
	assert.Equal(t, getArgString(argWithVal), "mystring required:false, type:string|list")

	argWithoutVal := asp.Argument{
		Name: "name",
		Type: []string{"string"},
	}
	assert.Equal(t, getArgString(argWithoutVal), "name required:true, type:string")
}

func getStatementByName(statements []*asp.Statement, name string) *asp.Statement {
	for _, stmt := range statements {
		if stmt.FuncDef != nil && stmt.FuncDef.Name == name {
			return stmt
		}
	}
	return nil
}
