package langserver

import (
	"core"
	"context"
	"path"
	"testing"

	"parse/asp"
	"tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestNewAnalyzer(t *testing.T) {
	a := newAnalyzer()
	assert.NotEqual(t, nil, a.BuiltIns)
}

func TestGetStatementsFromFile(t *testing.T) {
	a := newAnalyzer()
	core.FindRepoRoot()
	ctx := context.Background()
	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/BUILD")
	uri := lsp.DocumentURI("file://" + filepath)
	fileContent, err := ReadFile(ctx, uri)

	stmt, err := a.IdentFromFile(uri, fileContent)
	fmt.Println(err)
	fmt.Println("Luna", stmt[0].Action.Call.Arguments[0].Value.Val)
	fmt.Println(stmt)

}

func TestNewRuleDef(t *testing.T) {
	a := newAnalyzer()
	// Test header the definition for build_rule
	buildRule := a.parser.GetAllBuiltinStatements()[0].FuncDef
	expected := "def build_rule(name:str, cmd:str|dict='', test_cmd:str|dict='', srcs:list|dict=None, data:list=None, \n" +
		"               outs:list|dict=None, deps:list=None, exported_deps:list=None, secrets:list=None, \n" +
		"               tools:list|dict=None, labels:list=None, visibility:list=CONFIG.DEFAULT_VISIBILITY, \n" +
		"               hashes:list=None, binary:bool=False, test:bool=False, test_only:bool=CONFIG.DEFAULT_TESTONLY, \n" +
		"               building_description:str=None, needs_transitive_deps:bool=False, output_is_complete:bool=False, \n" +
		"               container:bool|dict=False, sandbox:bool=CONFIG.BUILD_SANDBOX, test_sandbox:bool=CONFIG.TEST_SANDBOX, \n" +
		"               no_test_output:bool=False, flaky:bool|int=0, build_timeout:int=0, test_timeout:int=0, \n" +
		"               pre_build:function=None, post_build:function=None, requires:list=None, \n" +
		"               provides:dict=None, licences:list=CONFIG.DEFAULT_LICENCES, test_outputs:list=None, \n" +
		"               system_srcs:list=None, stamp:bool=False, tag:str='', optional_outs:list=None, \n" +
		"               progress:bool=False, _urls:list=None)"
	ruledef := newRuleDef(buildRule)
	assert.Equal(t, expected, ruledef.Header)
	assert.Equal(t, len(ruledef.Arguments), len(ruledef.ArgMap))
	assert.Equal(t, false, ruledef.ArgMap["test_cmd"].required)
	assert.Equal(t, true, ruledef.ArgMap["name"].required)

	// Test header for len()
	lenFunc := a.parser.GetAllBuiltinStatements()[1].FuncDef
	expected = "def len(obj)"
	ruledef = newRuleDef(lenFunc)
	assert.Equal(t, expected, ruledef.Header)
	assert.Equal(t, 1, len(ruledef.ArgMap))
	assert.Equal(t, true, ruledef.ArgMap["obj"].required)

	// Test header for a string function, startswith()
	startswith := a.parser.GetAllBuiltinStatements()[9].FuncDef
	expected = "str.startswith(s:str)"
	ruledef = newRuleDef(startswith)
	assert.Equal(t, expected, ruledef.Header)
	assert.Equal(t, 1, len(ruledef.ArgMap))
	assert.Equal(t, true, ruledef.ArgMap["s"].required)

	// Test header for a config function, setdefault()
	setDefault := a.parser.GetAllBuiltinStatements()[37].FuncDef
	expected = "config.setdefault(key:str, default=None)"
	ruledef = newRuleDef(setDefault)
	assert.Equal(t, expected, ruledef.Header)
	assert.Equal(t, 2, len(ruledef.ArgMap))
	assert.Equal(t, false, ruledef.ArgMap["default"].required)
}

func TestGetArgument(t *testing.T) {
	argWithVal := asp.Argument{
		Name: "mystring",
		Type: []string{"string", "list"},
		Value: &asp.Expression{
			Optimised: &asp.OptimisedExpression{
				Local: "None",
			},
		},
	}
	assert.Equal(t, getArgument(argWithVal), "mystring:string|list=None")

	argWithoutVal := asp.Argument{
		Name: "name",
		Type: []string{"string"},
	}
	assert.Equal(t, getArgument(argWithoutVal), "name:string")
}
