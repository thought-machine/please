package langserver

import (
	"context"
	"core"
	"parse/asp"
	"parse/rules"
	"path"
	"path/filepath"
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

	// Check for methods map
	_, ok := a.Attributes["str"]
	assert.True(t, ok)
}

func TestAspStatementFromFile(t *testing.T) {
	a := newAnalyzer()

	filePath := "tools/build_langserver/langserver/test_data/example.build"
	uri := lsp.DocumentURI("file://" + filePath)

	stmts, err := a.AspStatementFromFile(uri)
	assert.Equal(t, err, nil)
	assert.Equal(t, stmts[0].Ident.Name, "go_library")

	assert.Equal(t, stmts[1].Ident.Name, "go_test")
}

func TestStatementFromPos(t *testing.T) {
	a := newAnalyzer()

	filePath := "tools/build_langserver/langserver/test_data/example.build"
	uri := lsp.DocumentURI("file://" + filePath)

	stmt, err := a.StatementFromPos(uri, lsp.Position{Line: 2, Character: 13})
	assert.Equal(t, err, nil)
	assert.Equal(t, "call", stmt.Ident.Type)
	assert.Equal(t, "go_library", stmt.Ident.Name)
	assert.Equal(t, "name", stmt.Ident.Action.Call.Arguments[0].Name)

	// Test on blank Area
	stmt, err = a.StatementFromPos(uri, lsp.Position{Line: 18, Character: 50})
	assert.Equal(t, err, nil)
	assert.True(t, nil == stmt)

	// Test out of range
	stmt, err = a.StatementFromPos(uri, lsp.Position{Line: 100, Character: 50})
	assert.Equal(t, err, nil)
	assert.True(t, nil == stmt)
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

	// Test header for a string function, format()
	stmt = getStatementByName(statements, "format")
	ruleDef = newRuleDef(string(ruleContent), stmt)

	assert.Equal(t, ruleDef.Header, "str.format()")
	assert.Equal(t, len(ruleDef.ArgMap), 0)
	assert.Equal(t, ruleDef.Object, "str")

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

func TestBuildLabelFromString(t *testing.T) {
	a := newAnalyzer()
	ctx := context.Background()
	filePath := "tools/build_langserver/langserver/test_data/example.build"
	uri := lsp.DocumentURI("file://" + filePath)

	// Test case for regular and complete BuildLabel path
	label, err := a.BuildLabelFromString(ctx, core.RepoRoot, uri, "//third_party/go:jsonrpc2")
	expectedContent := "go_get(\n" +
		"    name = \"jsonrpc2\",\n" +
		"    get = \"github.com/sourcegraph/jsonrpc2\",\n" +
		"    revision = \"549eb959f029d014d623104d40ab966d159a92de\",\n" +
		")"
	assert.Equal(t, err, nil)
	assert.Equal(t, path.Join(core.RepoRoot, "third_party/go/BUILD"), label.Path)
	assert.Equal(t, "jsonrpc2", label.Name)
	assert.Equal(t, expectedContent, label.BuildDefContent)

	// Test case for relative BuildLabel path
	label, err = a.BuildLabelFromString(ctx, core.RepoRoot, uri, ":langserver")
	expectedContent = "go_library(\n" +
		"    name = \"langserver\",\n" +
		"    srcs = glob(\n" +
		"        [\"*.go\"],\n" +
		"        exclude = [\"*_test.go\"],\n" +
		"    ),\n" +
		"    visibility = [\"//tools/build_langserver/...\", \"//src/core\"],\n" +
		"    deps = [\n" +
		"        \"//src/core\",\n" +
		"        \"//src/fs\",\n" +
		"        \"//src/parse\",\n" +
		"        \"//src/parse/asp\",\n" +
		"        \"//src/parse/rules\",\n" +
		"        \"//third_party/go:jsonrpc2\",\n" +
		"        \"//third_party/go:logging\",\n" +
		"        \"//tools/build_langserver/lsp\",\n" +
		"    ],\n" +
		")"
	absPath, err := filepath.Abs(filePath)
	assert.Equal(t, err, nil)
	assert.Equal(t, absPath, label.Path)
	assert.Equal(t, "langserver", label.Name)
	assert.Equal(t, expectedContent, label.BuildDefContent)

	// Test case for Allsubpackage BuildLabels: "//src/parse/..."
	label, err = a.BuildLabelFromString(ctx, core.RepoRoot,
		uri, "//src/parse/...")
	assert.Equal(t, err, nil)
	assert.True(t, nil == label.BuildDef)
	assert.Equal(t, "BuildLabel includes all subpackages in path: "+path.Join(core.RepoRoot, "src/parse"),
		label.BuildDefContent)

	// Test case for All targets in a BUILD file: "//src/parse:all"
	label, err = a.BuildLabelFromString(ctx, core.RepoRoot,
		uri, "//src/parse:all")
	assert.Equal(t, err, nil)
	assert.True(t, nil == label.BuildDef)
	assert.Equal(t, "BuildLabel includes all BuildTargets in BUILD file: "+path.Join(core.RepoRoot, "src/parse/BUILD"),
		label.BuildDefContent)

	// Test case for shortended BuildLabel
	label, err = a.BuildLabelFromString(ctx, core.RepoRoot,
		uri, "//src/core")
	assert.Equal(t, err, nil)

	label2, err := a.BuildLabelFromString(ctx, core.RepoRoot,
		uri, "//src/core:core")
	assert.Equal(t, err, nil)

	assert.Equal(t, label.BuildDefContent, label2.BuildDefContent)

	// Test case for subrepo
	label, err = a.BuildLabelFromString(ctx, core.RepoRoot,
		uri, "@mysubrepo//spam/eggs:ham")
	assert.Equal(t, err, nil)
	assert.True(t, nil == label.BuildDef)
	assert.Equal(t, "Subrepo label: @mysubrepo//spam/eggs:ham", label.BuildDefContent)
}

func TestAnalyzer_BuildDefFromUri(t *testing.T) {
	buildDefs, err := analyzer.BuildDefFromUri(exampleBuildURI)
	assert.Equal(t, err, nil)
	assert.Equal(t, 4, len(buildDefs))
	assert.Equal(t, []string{"//tools/build_langserver/...", "//src/core"}, buildDefs["langserver"].Visibility)
	t.Log(buildDefs["langserver_test"].Visibility)

	exampleBuildURI2 := lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/example2.build")
	buildDefs, err = analyzer.BuildDefFromUri(exampleBuildURI2)
	assert.Equal(t, 2, len(buildDefs))
	assert.Equal(t, []string{"PUBLIC"}, buildDefs["langserver_test"].Visibility)
}

/************************
 * Helper functions
 ************************/
func getStatementByName(statements []*asp.Statement, name string) *asp.Statement {
	for _, stmt := range statements {
		if stmt.FuncDef != nil && stmt.FuncDef.Name == name {
			return stmt
		}
	}
	return nil
}
