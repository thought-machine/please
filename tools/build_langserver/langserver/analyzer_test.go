package langserver

import (
	"context"

	"path"
	"path/filepath"
	"testing"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/src/parse/rules"
	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
	"strings"
)

func TestNewAnalyzer(t *testing.T) {
	a, err := newAnalyzer()
	assert.Equal(t, err, nil)

	assert.NotEqual(t, nil, a.BuiltIns)

	goLibrary := a.BuiltIns["go_library"]
	assert.Equal(t, 15, len(goLibrary.ArgMap))
	assert.Equal(t, true, goLibrary.ArgMap["name"].Required)

	// check preloadBuildDefs has being loaded
	goBinData := a.BuiltIns["go_bindata"]
	assert.Equal(t, 10, len(goBinData.ArgMap))
	assert.Equal(t, true, goBinData.ArgMap["name"].Required)
	assert.Equal(t, "input_dir=None", goBinData.ArgMap["input_dir"].Repr)

	// Ensure private funcDefs are not loaded
	for name := range a.BuiltIns {
		assert.False(t, strings.HasPrefix(name, "_"))
	}

	// Check for methods map
	_, ok := a.Attributes["str"]
	assert.True(t, ok)
}

func TestAspStatementFromFile(t *testing.T) {
	a, err := newAnalyzer()
	assert.Equal(t, err, nil)

	filePath := "tools/build_langserver/langserver/test_data/example.build"
	a.State.Config.Parse.BuildFileName = append(a.State.Config.Parse.BuildFileName, "example.build")
	uri := lsp.DocumentURI("file://" + filePath)

	stmts, err := a.AspStatementFromFile(uri)
	assert.Equal(t, err, nil)
	assert.Equal(t, stmts[0].Ident.Name, "go_library")

	assert.Equal(t, stmts[1].Ident.Name, "go_test")
}

func TestNewRuleDef(t *testing.T) {
	a, err := newAnalyzer()
	assert.Equal(t, err, nil)

	// Test header the definition for build_rule
	expected := "def go_library(name:str, srcs:list, asm_srcs:list=None, hdrs:list=None, out:str=None, deps:list=[],\n" +
		"               visibility:list=None, test_only:bool&testonly=False, complete:bool=True, cover:bool=True,\n" +
		"               filter_srcs:bool=True)"

	ruleContent := rules.MustAsset("go_rules.build_defs")

	statements, err := a.parser.ParseData(ruleContent, "go_rules.build_defs")
	assert.Equal(t, err, nil)

	stmt := getStatementByName(statements, "go_library")
	ruleDef := newRuleDef(string(ruleContent), stmt)

	assert.Equal(t, ruleDef.Header, expected)
	assert.Equal(t, len(ruleDef.Arguments), len(ruleDef.ArgMap))
	assert.Equal(t, false, ruleDef.ArgMap["_link_private"].Required)
	assert.Equal(t, true, ruleDef.ArgMap["name"].Required)
	assert.Equal(t, ruleDef.ArgMap["visibility"].Definition,
		"visibility required:false, type:list")
	assert.Equal(t, ruleDef.ArgMap["visibility"].Argument.Name, "visibility")
	assert.Equal(t, ruleDef.ArgMap["name"].Argument.Name, "name")
	assert.Equal(t, ruleDef.ArgMap["_link_private"].Argument.Name, "_link_private")

	// Test header for len()
	ruleContent = rules.MustAsset("builtins.build_defs")

	statements, err = a.parser.ParseData(ruleContent, "builtins.build_defs")
	assert.Equal(t, err, nil)

	stmt = getStatementByName(statements, "len")
	ruleDef = newRuleDef(string(ruleContent), stmt)

	assert.Equal(t, ruleDef.Header, "def len(obj:list|dict|str)")
	assert.Equal(t, len(ruleDef.ArgMap), 1)
	assert.Equal(t, true, ruleDef.ArgMap["obj"].Required)
	assert.Equal(t, ruleDef.ArgMap["obj"].Definition, "obj required:true, type:list|dict|str")
	assert.Equal(t, "obj:list|dict|str", ruleDef.ArgMap["obj"].Repr)
	assert.Equal(t, ruleDef.ArgMap["obj"].Argument.Name, "obj")

	// Test header for a string function, startswith()
	stmt = getStatementByName(statements, "startswith")
	ruleDef = newRuleDef(string(ruleContent), stmt)

	assert.Equal(t, ruleDef.Header, "str.startswith(s:str)")
	assert.Equal(t, len(ruleDef.ArgMap), 1)
	assert.Equal(t, true, ruleDef.ArgMap["s"].Required)
	assert.Equal(t, "s:str", ruleDef.ArgMap["s"].Repr)

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
	assert.Equal(t, false, ruleDef.ArgMap["default"].Required)
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
	a, err := newAnalyzer()
	assert.Equal(t, err, nil)

	ctx := context.Background()
	filePath := "tools/build_langserver/langserver/test_data/example.build"
	uri := lsp.DocumentURI("file://" + filePath)

	a.State.Config.Parse.BuildFileName = append(a.State.Config.Parse.BuildFileName, "example.build")

	// Test case for regular and complete BuildLabel path
	label, err := a.BuildLabelFromString(ctx, uri, "//third_party/go:jsonrpc2")
	expectedContent := "go_get(\n" +
		"    name = \"jsonrpc2\",\n" +
		"    get = \"github.com/sourcegraph/jsonrpc2\",\n" +
		"    revision = \"549eb959f029d014d623104d40ab966d159a92de\",\n" +
		")"
	assert.Equal(t, err, nil)
	assert.Equal(t, path.Join(core.RepoRoot, "third_party/go/BUILD"), label.Path)
	assert.Equal(t, "jsonrpc2", label.Name)
	assert.Equal(t, expectedContent, label.BuildDef.Content)
	assert.Equal(t, `BUILD Label: //third_party/go:jsonrpc2`, label.Definition)

	// Test case for relative BuildLabel path
	label, err = a.BuildLabelFromString(ctx, uri, ":langserver")
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
	assert.Equal(t, expectedContent, label.BuildDef.Content)

	// Test case for Allsubpackage BuildLabels: "//src/parse/..."
	label, err = a.BuildLabelFromString(ctx,
		uri, "//src/parse/...")
	assert.Equal(t, err, nil)
	assert.True(t, nil == label.BuildDef)
	assert.Equal(t, "BuildLabel includes all subpackages in path: "+path.Join(core.RepoRoot, "src/parse"),
		label.Definition)

	// Test case for All targets in a BUILD file: "//src/parse:all"
	label, err = a.BuildLabelFromString(ctx,
		uri, "//src/parse:all")
	assert.Equal(t, err, nil)
	assert.True(t, nil == label.BuildDef)
	assert.Equal(t, "BuildLabel includes all BuildTargets in BUILD file: "+path.Join(core.RepoRoot, "src/parse/BUILD"),
		label.Definition)

	// Test case for shortended BuildLabel
	label, err = a.BuildLabelFromString(ctx, uri, "//src/core")
	assert.Equal(t, err, nil)

	label2, err := a.BuildLabelFromString(ctx, uri, "//src/core:core")
	assert.Equal(t, err, nil)

	assert.Equal(t, label.Definition, label2.Definition)

	// Test case for subrepo
	label, err = a.BuildLabelFromString(ctx, uri, "@mysubrepo//spam/eggs:ham")
	assert.Equal(t, err, nil)
	assert.True(t, nil == label.BuildDef)
	assert.Equal(t, "Subrepo label: @mysubrepo//spam/eggs:ham", label.Definition)
}

func TestAnalyzer_BuildLabelFromStringBogusLabel(t *testing.T) {
	a, err := newAnalyzer()
	assert.Equal(t, err, nil)

	ctx := context.Background()

	// Ensure we get an error when we pass in a bogus label
	label, err := a.BuildLabelFromString(ctx, exampleBuildURI, "//blah/foo")
	assert.NotEqual(t, err, nil)
	assert.True(t, nil == label)

	label, err = a.BuildLabelFromString(ctx, exampleBuildURI, "//src/core:blah")
	assert.NotEqual(t, err, nil)
	assert.True(t, nil == label)
}

func TestAnalyzer_BuildDefFromUri(t *testing.T) {
	ctx := context.Background()

	buildDefs, err := analyzer.BuildDefsFromURI(ctx, exampleBuildURI)
	assert.Equal(t, err, nil)
	assert.Equal(t, 6, len(buildDefs))
	assert.True(t, StringInSlice(buildDefs["langserver"].Visibility, "//tools/build_langserver/..."))
	assert.True(t, StringInSlice(buildDefs["langserver"].Visibility, "//src/core"))
	t.Log(buildDefs["langserver_test"].Visibility)

	exampleBuildURI2 := lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/example2.build")
	buildDefs, err = analyzer.BuildDefsFromURI(ctx, exampleBuildURI2)
	assert.Equal(t, 2, len(buildDefs))
	assert.True(t, StringInSlice(buildDefs["langserver_test"].Visibility, "PUBLIC"))
}

func TestAnalyzer_IsBuildFile(t *testing.T) {
	a, err := newAnalyzer()
	assert.Equal(t, err, nil)

	uri := lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/example.build")

	assert.False(t, a.IsBuildFile(uri))

	a.State.Config.Parse.BuildFileName = append(a.State.Config.Parse.BuildFileName, "example.build")
	assert.True(t, a.IsBuildFile(uri))
}

func TestAnalyzer_VariableFromContentGLOBAL(t *testing.T) {
	a, err := newAnalyzer()
	assert.Equal(t, err, nil)
	pos := &lsp.Position{Line: 5, Character: 0}

	// Tests for string variables
	vars := a.VariablesFromContent(`my_str = "my str"`+"   \n"+
		`another_str = ""`+"\n   "+`more_empty = ''`, pos)
	assert.Equal(t, "my_str", vars["my_str"].Name)
	assert.Equal(t, "another_str", vars["another_str"].Name)
	assert.Equal(t, "more_empty", vars["more_empty"].Name)
	for _, v := range vars {
		assert.Equal(t, "str", v.Type)
	}

	vars = a.VariablesFromContent(`fstr = f"blah"`, pos)
	assert.Equal(t, "str", vars["fstr"].Type)

	// Tests for int variables
	vars = a.VariablesFromContent(`my_int = 34`, pos)
	assert.Equal(t, "my_int", vars["my_int"].Name)
	assert.Equal(t, "int", vars["my_int"].Type)

	// Tests for list variables
	vars = a.VariablesFromContent(`my_list = []`, pos)
	assert.Equal(t, "my_list", vars["my_list"].Name)
	assert.Equal(t, "list", vars["my_list"].Type)

	vars = a.VariablesFromContent(`my_list2 = [1, 2, 3]`, pos)
	assert.Equal(t, "my_list2", vars["my_list2"].Name)
	assert.Equal(t, "list", vars["my_list2"].Type)

	// Tests for dict variables
	vars = a.VariablesFromContent(`my_dict = {'foo': 1, 'bar': 3}`, pos)
	assert.Equal(t, "my_dict", vars["my_dict"].Name)
	assert.Equal(t, "dict", vars["my_dict"].Type)

	// Test for calls
	vars = a.VariablesFromContent(`my_call = go_library()`, pos)
	assert.Equal(t, "", vars["my_call"].Type)

	// Test for reassigning variable
	vars = a.VariablesFromContent(`foo = "hello"`+"\n"+`foo = 90`, pos)
	assert.Equal(t, "int", vars["foo"].Type)
}

func TestAnalyzer_GetSubinclude(t *testing.T) {
	a, err := newAnalyzer()
	assert.Equal(t, err, nil)
	ctx := context.Background()

	stmts, err := a.AspStatementFromFile(subincludeURI)
	assert.NoError(t, err)

	subinclude := a.GetSubinclude(ctx, stmts, subincludeURI)
	assert.Equal(t, len(subinclude), 1)
	_, ok := subinclude["plz_e2e_test"]
	assert.True(t, ok)
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
