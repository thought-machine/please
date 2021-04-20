package asp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBasic(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/basic.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))
	assert.NotNil(t, statements[0].FuncDef)
	assert.Equal(t, "test", statements[0].FuncDef.Name)
	assert.Equal(t, 1, len(statements[0].FuncDef.Arguments))
	assert.Equal(t, "x", statements[0].FuncDef.Arguments[0].Name)
	assert.Equal(t, 1, len(statements[0].FuncDef.Statements))
	assert.True(t, statements[0].FuncDef.Statements[0].Pass)

	// Test for Endpos
	assert.Equal(t, 9, statements[0].EndPos.Column)
	assert.Equal(t, 3, statements[0].EndPos.Line)
}

func TestParseDefaultArguments(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/default_arguments.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))
	assert.NotNil(t, statements[0].FuncDef)
	assert.Equal(t, "test", statements[0].FuncDef.Name)
	assert.Equal(t, 3, len(statements[0].FuncDef.Arguments))
	assert.Equal(t, 1, len(statements[0].FuncDef.Statements))
	assert.True(t, statements[0].FuncDef.Statements[0].Pass)

	args := statements[0].FuncDef.Arguments
	assert.Equal(t, "name", args[0].Name)
	assert.Equal(t, "\"name\"", args[0].Value.Val.String)
	assert.Equal(t, "timeout", args[1].Name)
	assert.Equal(t, 10, args[1].Value.Val.Int.Int)
	assert.Equal(t, "args", args[2].Name)
	assert.Equal(t, "None", args[2].Value.Val.Bool)

	// Test for Endpos
	assert.Equal(t, 9, statements[0].EndPos.Column)
	assert.Equal(t, 3, statements[0].EndPos.Line)
}

func TestParseFunctionCalls(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/function_call.build")
	assert.NoError(t, err)
	assert.Equal(t, 5, len(statements))

	assert.NotNil(t, statements[0].Ident.Action.Call)
	assert.Equal(t, "package", statements[0].Ident.Name)
	assert.Equal(t, 0, len(statements[0].Ident.Action.Call.Arguments))

	assert.NotNil(t, statements[2].Ident.Action.Call)
	assert.Equal(t, "package", statements[2].Ident.Name)
	assert.Equal(t, 1, len(statements[2].Ident.Action.Call.Arguments))
	arg := statements[2].Ident.Action.Call.Arguments[0]
	assert.Equal(t, "default_visibility", arg.Name)
	assert.NotNil(t, arg.Value)
	assert.Equal(t, 1, len(arg.Value.Val.List.Values))
	assert.Equal(t, "\"PUBLIC\"", arg.Value.Val.List.Values[0].Val.String)

	assert.NotNil(t, statements[3].Ident.Action.Call)
	assert.Equal(t, "python_library", statements[3].Ident.Name)
	assert.Equal(t, 2, len(statements[3].Ident.Action.Call.Arguments))
	args := statements[3].Ident.Action.Call.Arguments
	assert.Equal(t, "name", args[0].Name)
	assert.Equal(t, "\"lib\"", args[0].Value.Val.String)
	assert.Equal(t, "srcs", args[1].Name)
	assert.NotNil(t, args[1].Value.Val.List)
	assert.Equal(t, 2, len(args[1].Value.Val.List.Values))
	assert.Equal(t, "\"lib1.py\"", args[1].Value.Val.List.Values[0].Val.String)
	assert.Equal(t, "\"lib2.py\"", args[1].Value.Val.List.Values[1].Val.String)

	assert.NotNil(t, statements[4].Ident.Action.Call)
	assert.Equal(t, "subinclude", statements[4].Ident.Name)
	assert.NotNil(t, statements[4].Ident.Action.Call)
	assert.Equal(t, 1, len(statements[4].Ident.Action.Call.Arguments))
	assert.Equal(t, "\"//build_defs:version\"", statements[4].Ident.Action.Call.Arguments[0].Value.Val.String)

	// Test for Endpos
	assert.Equal(t, 10, statements[0].EndPos.Column)
	assert.Equal(t, 1, statements[0].EndPos.Line)
	assert.Equal(t, 10, statements[1].EndPos.Column)
	assert.Equal(t, 2, statements[1].EndPos.Line)
	assert.Equal(t, 41, statements[2].EndPos.Column)
	assert.Equal(t, 3, statements[2].EndPos.Line)
	assert.Equal(t, 2, statements[3].EndPos.Column)
	assert.Equal(t, 11, statements[3].EndPos.Line)
	assert.Equal(t, 35, statements[4].EndPos.Column)
	assert.Equal(t, 13, statements[4].EndPos.Line)
}

func TestParseAssignments(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/assignments.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	assert.NotNil(t, statements[0].Ident.Action.Assign)
	assert.Equal(t, "x", statements[0].Ident.Name)
	ass := statements[0].Ident.Action.Assign.Val.Dict
	assert.NotNil(t, ass)
	assert.Equal(t, 3, len(ass.Items))
	assert.Equal(t, "\"mickey\"", ass.Items[0].Key.Val.String)
	assert.Equal(t, 3, ass.Items[0].Value.Val.Int.Int)
	assert.Equal(t, "\"donald\"", ass.Items[1].Key.Val.String)
	assert.Equal(t, "\"sora\"", ass.Items[1].Value.Val.String)
	assert.Equal(t, "\"goofy\"", ass.Items[2].Key.Val.String)
	assert.Equal(t, "riku", ass.Items[2].Value.Val.Ident.Name)

	// Test for Endpos
	assert.Equal(t, 2, statements[0].EndPos.Column)
	assert.Equal(t, 5, statements[0].EndPos.Line)
}

func TestForStatement(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/for_statement.build")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(statements))

	assert.NotNil(t, statements[0].Ident.Action.Assign)
	assert.Equal(t, "LANGUAGES", statements[0].Ident.Name)
	assert.Equal(t, 2, len(statements[0].Ident.Action.Assign.Val.List.Values))

	assert.NotNil(t, statements[1].For)
	assert.Equal(t, []string{"language"}, statements[1].For.Names)
	assert.Equal(t, "LANGUAGES", statements[1].For.Expr.Val.Ident.Name)
	assert.Equal(t, 2, len(statements[1].For.Statements))

	// Test for Endpos
	assert.Equal(t, 2, statements[0].EndPos.Column)
	assert.Equal(t, 4, statements[0].EndPos.Line)
	assert.Equal(t, 6, statements[1].EndPos.Column)
	assert.Equal(t, 12, statements[1].EndPos.Line)
}

func TestOperators(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/operators.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	assert.NotNil(t, statements[0].Ident.Action.Call)
	assert.Equal(t, "genrule", statements[0].Ident.Name)
	assert.Equal(t, 2, len(statements[0].Ident.Action.Call.Arguments))

	arg := statements[0].Ident.Action.Call.Arguments[1]
	assert.Equal(t, "srcs", arg.Name)
	assert.NotNil(t, arg.Value.Val.List)
	assert.Equal(t, 1, len(arg.Value.Val.List.Values))
	assert.Equal(t, "\"//something:test_go\"", arg.Value.Val.List.Values[0].Val.String)
	assert.Equal(t, 1, len(arg.Value.Op))
	assert.Equal(t, Add, arg.Value.Op[0].Op)
	call := arg.Value.Op[0].Expr.Val.Ident.Action[0].Call
	assert.Equal(t, "glob", arg.Value.Op[0].Expr.Val.Ident.Name)
	assert.NotNil(t, call)
	assert.Equal(t, 1, len(call.Arguments))
	assert.NotNil(t, call.Arguments[0].Value.Val.List)
	assert.Equal(t, 1, len(call.Arguments[0].Value.Val.List.Values))
	assert.Equal(t, "\"*.go\"", call.Arguments[0].Value.Val.List.Values[0].Val.String)

	// Test for Endpos
	assert.Equal(t, 2, statements[0].EndPos.Column)
	assert.Equal(t, 4, statements[0].EndPos.Line)
}

func TestIndexing(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/indexing.build")
	assert.NoError(t, err)
	assert.Equal(t, 7, len(statements))

	assert.Equal(t, "x", statements[0].Ident.Name)
	assert.NotNil(t, statements[0].Ident.Action.Assign)
	assert.Equal(t, "\"test\"", statements[0].Ident.Action.Assign.Val.String)

	assert.Equal(t, "y", statements[1].Ident.Name)
	assert.NotNil(t, statements[1].Ident.Action.Assign)
	assert.Equal(t, "x", statements[1].Ident.Action.Assign.Val.Ident.Name)
	assert.Equal(t, 1, len(statements[1].Ident.Action.Assign.Val.Slices))
	assert.Equal(t, 2, statements[1].Ident.Action.Assign.Val.Slices[0].Start.Val.Int.Int)
	assert.Equal(t, "", statements[1].Ident.Action.Assign.Val.Slices[0].Colon)
	assert.Nil(t, statements[1].Ident.Action.Assign.Val.Slices[0].End)

	assert.Equal(t, "z", statements[2].Ident.Name)
	assert.NotNil(t, statements[2].Ident.Action.Assign)
	assert.Equal(t, "x", statements[2].Ident.Action.Assign.Val.Ident.Name)
	assert.Equal(t, 1, len(statements[2].Ident.Action.Assign.Val.Slices))
	assert.Equal(t, 1, statements[2].Ident.Action.Assign.Val.Slices[0].Start.Val.Int.Int)
	assert.Equal(t, ":", statements[2].Ident.Action.Assign.Val.Slices[0].Colon)
	assert.Equal(t, -1, statements[2].Ident.Action.Assign.Val.Slices[0].End.Val.Int.Int)

	assert.Equal(t, "a", statements[3].Ident.Name)
	assert.NotNil(t, statements[3].Ident.Action.Assign)
	assert.Equal(t, "x", statements[3].Ident.Action.Assign.Val.Ident.Name)
	assert.Equal(t, 1, len(statements[3].Ident.Action.Assign.Val.Slices))
	assert.Equal(t, 2, statements[3].Ident.Action.Assign.Val.Slices[0].Start.Val.Int.Int)
	assert.Equal(t, ":", statements[3].Ident.Action.Assign.Val.Slices[0].Colon)
	assert.Nil(t, statements[3].Ident.Action.Assign.Val.Slices[0].End)

	assert.Equal(t, "b", statements[4].Ident.Name)
	assert.NotNil(t, statements[4].Ident.Action.Assign)
	assert.Equal(t, "x", statements[4].Ident.Action.Assign.Val.Ident.Name)
	assert.Equal(t, 1, len(statements[4].Ident.Action.Assign.Val.Slices))
	assert.Nil(t, statements[4].Ident.Action.Assign.Val.Slices[0].Start)
	assert.Equal(t, ":", statements[4].Ident.Action.Assign.Val.Slices[0].Colon)
	assert.Equal(t, 2, statements[4].Ident.Action.Assign.Val.Slices[0].End.Val.Int.Int)

	assert.Equal(t, "c", statements[5].Ident.Name)
	assert.NotNil(t, statements[5].Ident.Action.Assign)
	assert.Equal(t, "x", statements[5].Ident.Action.Assign.Val.Ident.Name)
	assert.Equal(t, 1, len(statements[5].Ident.Action.Assign.Val.Slices))
	assert.Equal(t, "y", statements[5].Ident.Action.Assign.Val.Slices[0].Start.Val.Ident.Name)
	assert.Equal(t, "", statements[5].Ident.Action.Assign.Val.Slices[0].Colon)
	assert.Nil(t, statements[5].Ident.Action.Assign.Val.Slices[0].End)

	// Test for Endpos
	assert.Equal(t, 11, statements[0].EndPos.Column)
	assert.Equal(t, 1, statements[0].EndPos.Line)
	assert.Equal(t, 9, statements[1].EndPos.Column)
	assert.Equal(t, 3, statements[1].EndPos.Line)
	assert.Equal(t, 12, statements[2].EndPos.Column)
	assert.Equal(t, 5, statements[2].EndPos.Line)
	assert.Equal(t, 10, statements[3].EndPos.Column)
	assert.Equal(t, 7, statements[3].EndPos.Line)
	assert.Equal(t, 10, statements[4].EndPos.Column)
	assert.Equal(t, 9, statements[4].EndPos.Line)
	assert.Equal(t, 9, statements[5].EndPos.Column)
	assert.Equal(t, 11, statements[5].EndPos.Line)
}

func TestIfStatement(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/if_statement.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	ifs := statements[0].If
	assert.NotNil(t, ifs)
	assert.Equal(t, "condition_a", ifs.Condition.Val.Ident.Name)
	assert.Equal(t, And, ifs.Condition.Op[0].Op)
	assert.Equal(t, "condition_b", ifs.Condition.Op[0].Expr.Val.Ident.Name)
	assert.Equal(t, 1, len(ifs.Statements))
	assert.Equal(t, "genrule", ifs.Statements[0].Ident.Name)

	// Test for Endpos
	assert.Equal(t, 6, statements[0].EndPos.Column)
	assert.Equal(t, 4, statements[0].EndPos.Line)
}

func TestDoubleUnindent(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/double_unindent.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	assert.NotNil(t, statements[0].For)
	assert.Equal(t, "y", statements[0].For.Names[0])
	assert.Equal(t, "x", statements[0].For.Expr.Val.Ident.Name)
	assert.Equal(t, 1, len(statements[0].For.Statements))

	for2 := statements[0].For.Statements[0].For
	assert.NotNil(t, for2)
	assert.Equal(t, "z", for2.Names[0])
	assert.Equal(t, "y", for2.Expr.Val.Ident.Name)
	assert.Equal(t, 1, len(for2.Statements))
	assert.Equal(t, "genrule", for2.Statements[0].Ident.Name)

	// Test for Endpos
	assert.Equal(t, 10, statements[0].EndPos.Column)
	assert.Equal(t, 5, statements[0].EndPos.Line)
}

func TestInlineIf(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/inline_if.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	assert.Equal(t, "x", statements[0].Ident.Name)
	ass := statements[0].Ident.Action.Assign
	assert.NotNil(t, ass)
	assert.NotNil(t, ass.Val.List)
	assert.Equal(t, 1, len(ass.Val.List.Values))
	assert.NotNil(t, ass.If)
	assert.Equal(t, "y", ass.If.Condition.Val.Ident.Name)
	assert.EqualValues(t, Is, ass.If.Condition.Op[0].Op)
	assert.Equal(t, "None", ass.If.Condition.Op[0].Expr.Val.Bool)
	assert.NotNil(t, ass.If.Else.Val.List)
	assert.Equal(t, 1, len(ass.If.Else.Val.List.Values))

	// Test for Endpos
	assert.Equal(t, 1, statements[0].EndPos.Line)
	assert.Equal(t, 38, statements[0].EndPos.Column)
}

func TestFunctionDef(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/function_def.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))
	assert.Equal(t, 3, len(statements[0].FuncDef.Statements))
	assert.Equal(t, "Generate a C or C++ library target.", stringLiteral(statements[0].FuncDef.Docstring))

	// Test for Endpos
	assert.Equal(t, 9, statements[0].EndPos.Line)
	assert.Equal(t, 22, statements[0].EndPos.Column)
}

func TestComprehension(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/comprehension.build")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(statements))

	assert.NotNil(t, statements[0].Ident.Action.Assign)
	assert.NotNil(t, statements[1].Ident.Action.Assign)
	assert.Equal(t, 1, len(statements[0].Ident.Action.Assign.Val.List.Values))
	assert.NotNil(t, statements[0].Ident.Action.Assign.Val.List.Comprehension)
	assert.NotNil(t, statements[1].Ident.Action.Assign.Val.Dict.Comprehension)

	// Test for Endpos
	assert.Equal(t, 1, statements[0].EndPos.Line)
	assert.Equal(t, 29, statements[0].EndPos.Column)
	assert.Equal(t, 3, statements[1].EndPos.Line)
	assert.Equal(t, 47, statements[1].EndPos.Column)
}

func TestMethodsOnLiterals(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/literal_methods.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))

	// Test for Endpos
	assert.Equal(t, 5, statements[0].EndPos.Line)
	assert.Equal(t, 3, statements[0].EndPos.Column)
}

func TestUnaryOp(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/unary_op.build")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(statements))

	assert.NotNil(t, statements[0].Ident.Action.Assign.UnaryOp)
	assert.Equal(t, "-", statements[0].Ident.Action.Assign.UnaryOp.Op)
	assert.Equal(t, "len", statements[0].Ident.Action.Assign.UnaryOp.Expr.Ident.Name)
	assert.NotNil(t, statements[1].Ident.Action.Assign.UnaryOp)
	assert.Equal(t, "not", statements[1].Ident.Action.Assign.UnaryOp.Op)
	assert.Equal(t, "x", statements[1].Ident.Action.Assign.UnaryOp.Expr.Ident.Name)

	// Test for Endpos
	assert.Equal(t, 1, statements[0].EndPos.Line)
	assert.Equal(t, 19, statements[0].EndPos.Column)
	assert.Equal(t, 3, statements[1].EndPos.Line)
	assert.Equal(t, 10, statements[1].EndPos.Column)
	assert.Equal(t, 5, statements[2].EndPos.Line)
	assert.Equal(t, 24, statements[2].EndPos.Column)
}

func TestAugmentedAssignment(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/aug_assign.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))
	assert.NotNil(t, statements[0].Ident.Action.AugAssign)

	// Test for Endpos
	assert.Equal(t, 1, statements[0].EndPos.Line)
	assert.Equal(t, 17, statements[0].EndPos.Column)
}

func TestRaise(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/raise.build")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(statements))

	// Test for Endpos
	assert.Equal(t, 1, statements[0].EndPos.Line)
	assert.Equal(t, 27, statements[0].EndPos.Column)
	assert.Equal(t, 4, statements[1].EndPos.Line)
	assert.Equal(t, 31, statements[1].EndPos.Column)
}

func TestElseStatement(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/else.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))
	assert.NotNil(t, statements[0].If)
	assert.Equal(t, 1, len(statements[0].If.Statements))
	assert.Equal(t, 2, len(statements[0].If.Elif))
	assert.Equal(t, 1, len(statements[0].If.Elif[0].Statements))
	assert.NotNil(t, statements[0].If.Elif[0].Condition)
	assert.Equal(t, 1, len(statements[0].If.Elif[1].Statements))
	assert.NotNil(t, statements[0].If.Elif[1].Condition)
	assert.Equal(t, 1, len(statements[0].If.ElseStatements))

	// Test for Endpos
	assert.Equal(t, 8, statements[0].EndPos.Line)
	assert.Equal(t, 9, statements[0].EndPos.Column)
}

func TestDestructuringAssignment(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/destructuring_assign.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))
	assert.NotNil(t, statements[0].Ident)
	assert.Equal(t, "x", statements[0].Ident.Name)
	assert.NotNil(t, statements[0].Ident.Unpack)
	assert.Equal(t, 1, len(statements[0].Ident.Unpack.Names))
	assert.Equal(t, "y", statements[0].Ident.Unpack.Names[0])
	assert.NotNil(t, statements[0].Ident.Unpack.Expr)
	assert.Equal(t, "something", statements[0].Ident.Unpack.Expr.Val.Ident.Name)

	// Test for Endpos
	assert.Equal(t, 1, statements[0].EndPos.Line)
	assert.Equal(t, 19, statements[0].EndPos.Column)
}

func TestMultipleActions(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/multiple_action.build")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(statements))
	assert.NotNil(t, statements[0].Ident.Action.Assign)
	assert.Equal(t, "y", statements[0].Ident.Action.Assign.Val.Ident.Name)

	// Test for Endpos
	assert.Equal(t, 1, statements[0].EndPos.Line)
	assert.Equal(t, 64, statements[0].EndPos.Column)
	assert.Equal(t, 3, statements[1].EndPos.Line)
	assert.Equal(t, 14, statements[1].EndPos.Column)
	assert.Equal(t, 5, statements[2].EndPos.Line)
	assert.Equal(t, 12, statements[2].EndPos.Column)
}

func TestAssert(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/assert.build")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(statements))

	// Test for Endpos
	assert.Equal(t, 1, statements[0].EndPos.Line)
	assert.Equal(t, 74, statements[0].EndPos.Column)

	assert.Equal(t, 4, statements[1].EndPos.Line)
	assert.Equal(t, 18, statements[1].EndPos.Column)
}

func TestOptimise(t *testing.T) {
	p := newParser()
	statements, err := p.parse("src/parse/asp/test_data/optimise.build")
	assert.NoError(t, err)
	assert.Equal(t, 5, len(statements))
	statements = p.optimise(statements)
	assert.Equal(t, 4, len(statements))

	assert.NotNil(t, statements[0].FuncDef)
	assert.Equal(t, 0, len(statements[0].FuncDef.Statements))
	// Test for Endpos
	assert.Equal(t, 41, statements[0].EndPos.Column)
	assert.Equal(t, 4, statements[0].EndPos.Line)

	assert.NotNil(t, statements[1].FuncDef)
	assert.Equal(t, 0, len(statements[1].FuncDef.Statements))
	// Test for Endpos
	assert.Equal(t, 9, statements[1].EndPos.Column)
	assert.Equal(t, 7, statements[1].EndPos.Line)

	assert.NotNil(t, statements[2].FuncDef)
	assert.Equal(t, 1, len(statements[2].FuncDef.Statements))
	// Test for Endpos
	assert.Equal(t, 18, statements[2].EndPos.Column)
	assert.Equal(t, 10, statements[2].EndPos.Line)

	ident := statements[2].FuncDef.Statements[0].Ident
	assert.NotNil(t, ident)
	assert.Equal(t, "l", ident.Name)
	assert.NotNil(t, ident.Action.AugAssign)
	assert.NotNil(t, ident.Action.AugAssign.Val.List)

	// Test for Endpos
	assert.Equal(t, 11, statements[3].EndPos.Column)
	assert.Equal(t, 13, statements[3].EndPos.Line)
}

func TestMultilineStringQuotes(t *testing.T) {
	statements, err := newParser().parse("src/parse/asp/test_data/multiline_string_quotes.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))
	assert.NotNil(t, statements[0].Ident)
	assert.NotNil(t, statements[0].Ident.Action)
	assert.NotNil(t, statements[0].Ident.Action.Assign)
	expected := `"
#include "UnitTest++/UnitTest++.h"
"`
	assert.Equal(t, expected, statements[0].Ident.Action.Assign.Val.String)

	// TODO(BNM): It would be nice if we can get the actual EndPos for the multiline
	// assert.Equal(t, 4, statements[0].EndPos.Column)
	// assert.Equal(t, 3, statements[0].EndPos.Line)
}

func TestExample0(t *testing.T) {
	// These tests are specific examples that turned out to fail.
	statements, err := newParser().parse("src/parse/asp/test_data/example.build")
	assert.NoError(t, err)

	// Test for Endpos
	assert.Equal(t, 2, statements[0].EndPos.Column)
	assert.Equal(t, 14, statements[0].EndPos.Line)
	assert.Equal(t, 2, statements[1].EndPos.Column)
	assert.Equal(t, 24, statements[1].EndPos.Line)
	assert.Equal(t, 2, statements[2].EndPos.Column)
	assert.Equal(t, 34, statements[2].EndPos.Line)
	assert.Equal(t, 2, statements[3].EndPos.Column)
	assert.Equal(t, 43, statements[3].EndPos.Line)
}

func TestExample1(t *testing.T) {
	// These tests are specific examples that turned out to fail.
	stmts, err := newParser().parse("src/parse/asp/test_data/example_1.build")
	assert.NoError(t, err)

	// Test for Endpos
	assert.Equal(t, 13, stmts[0].EndPos.Column)
	assert.Equal(t, 6, stmts[0].EndPos.Line)
}

func TestExample2(t *testing.T) {
	stmts, err := newParser().parse("src/parse/asp/test_data/example_2.build")
	assert.NoError(t, err)

	// Test for Endpos
	assert.Equal(t, 35, stmts[0].EndPos.Column)
	assert.Equal(t, 1, stmts[0].EndPos.Line)
	assert.Equal(t, 2, stmts[1].EndPos.Column)
	assert.Equal(t, 7, stmts[1].EndPos.Line)
}

func TestExample3(t *testing.T) {
	stmts, err := newParser().parse("src/parse/asp/test_data/example_3.build")
	assert.NoError(t, err)

	// Test for Endpos
	assert.Equal(t, 2, stmts[0].EndPos.Column)
	assert.Equal(t, 4, stmts[0].EndPos.Line)

	// Test for IdentExpr.Endpos
	property := stmts[0].Ident.Action.Assign.Val.Ident.Action[0].Property
	assert.Equal(t, 1, property.Pos.Line)
	assert.Equal(t, 16, property.Pos.Column)
	assert.Equal(t, 4, property.EndPos.Line)
	assert.Equal(t, 2, property.EndPos.Column)
}

func TestExample4(t *testing.T) {
	stmts, err := newParser().parse("src/parse/asp/test_data/example_4.build")
	assert.NoError(t, err)
	assert.Equal(t, len(stmts), 1)

	// Test for Endpos
	assert.Equal(t, 1, stmts[0].EndPos.Line)
	assert.Equal(t, 38, stmts[0].EndPos.Column)
}

func TestExample5(t *testing.T) {
	stmts, err := newParser().parse("src/parse/asp/test_data/example_5.build")
	assert.NoError(t, err)

	// Test for Endpos
	assert.Equal(t, 1, stmts[0].EndPos.Line)
	assert.Equal(t, 68, stmts[0].EndPos.Column)
}

func TestExample6(t *testing.T) {
	stmts, err := newParser().parse("src/parse/asp/test_data/example_6.build")
	assert.NoError(t, err)

	// Test for Endpos
	assert.Equal(t, 1, stmts[0].EndPos.Line)
	assert.Equal(t, 80, stmts[0].EndPos.Column)

	// Test for IdentExpr.Endpos
	property := stmts[0].Ident.Action.Assign.Val.Property
	assert.Equal(t, 1, property.Pos.Line)
	assert.Equal(t, 15, property.Pos.Column)
	assert.Equal(t, 1, property.EndPos.Line)
	assert.Equal(t, 60, property.EndPos.Column)

	property = stmts[1].Ident.Action.Call.Arguments[0].Value.Val.Property
	assert.Equal(t, 4, property.Pos.Line)
	assert.Equal(t, 31, property.Pos.Column)
	assert.Equal(t, 4, property.EndPos.Line)
	assert.Equal(t, 66, property.EndPos.Column)
}

func TestPrecedence(t *testing.T) {
	stmts, err := newParser().parse("src/parse/asp/test_data/precedence.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(stmts))
	assert.NotNil(t, stmts[0].Ident.Action.Assign.If)

	// Test for Endpos
	assert.Equal(t, 1, stmts[0].EndPos.Line)
	assert.Equal(t, 35, stmts[0].EndPos.Column)
}

func TestMissingNewlines(t *testing.T) {
	stmts, err := newParser().parse("src/parse/asp/test_data/no_newline.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(stmts))

	// Test for Endpos
	assert.Equal(t, 1, stmts[0].EndPos.Line)
	assert.Equal(t, 10, stmts[0].EndPos.Column)
}

func TestRepeatedArguments(t *testing.T) {
	_, err := newParser().parse("src/parse/asp/test_data/repeated_arguments.build")
	assert.Error(t, err)
}

func TestConstantAssignments(t *testing.T) {
	_, err := newParser().parse("src/parse/asp/test_data/constant_assign.build")
	assert.Error(t, err)
}

func TestFStrings(t *testing.T) {
	stmts, err := newParser().parse("src/parse/asp/test_data/fstring.build")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(stmts))

	f := stmts[1].Ident.Action.Assign.Val.FString
	assert.NotNil(t, f)
	assert.Equal(t, "", f.Suffix)
	assert.Equal(t, 1, len(f.Vars))
	assert.Equal(t, "", f.Vars[0].Prefix)
	assert.Equal(t, "x", f.Vars[0].Var)

	f = stmts[2].Ident.Action.Assign.Val.FString
	assert.NotNil(t, f)
	assert.Equal(t, " fin", f.Suffix)
	assert.Equal(t, 2, len(f.Vars))
	assert.Equal(t, "x: ", f.Vars[0].Prefix)
	assert.Equal(t, "x", f.Vars[0].Var)
	assert.Equal(t, " y: ", f.Vars[1].Prefix)
	assert.Equal(t, "y", f.Vars[1].Var)

	// Test for Endpos
	assert.Equal(t, 1, stmts[0].EndPos.Line)
	assert.Equal(t, 8, stmts[0].EndPos.Column)
	assert.Equal(t, 2, stmts[1].EndPos.Line)
	assert.Equal(t, 11, stmts[1].EndPos.Column)
	assert.Equal(t, 3, stmts[2].EndPos.Line)
	assert.Equal(t, 25, stmts[2].EndPos.Column)
	assert.Equal(t, 6, stmts[3].EndPos.Line)
	assert.Equal(t, 15, stmts[3].EndPos.Column)
}

func TestFuncReturnTypes(t *testing.T) {
	stmts, err := newParser().parse("src/parse/asp/test_data/return_type.build")
	assert.NoError(t, err)

	assert.Equal(t, "str", stmts[0].FuncDef.Return)
	assert.Equal(t, "config", stmts[2].FuncDef.Return)
	assert.Equal(t, "dict", stmts[3].FuncDef.Return)
}

func TestFStringConcat(t *testing.T) {
	t.Run("lhs string, rhs fstring", func(t *testing.T) {
		lhs := &ValueExpression{
			String: "\"this is the left hand side\"",
		}

		rhs := &ValueExpression{
			FString: &FString{
				Vars: []struct {
					Prefix string
					Var    string
					Config string
				}{
					{
						Prefix: " this is the rhs: ",
						Var:    "rhs",
						Config: "",
					},
				},
				Suffix: " suffix",
			},
		}

		res := concatStrings(lhs, rhs)

		require.NotNil(t, res.FString)
		assert.Len(t, res.FString.Vars, 1)
		assert.Equal(t, "this is the left hand side this is the rhs: ", res.FString.Vars[0].Prefix)
		assert.Equal(t, "rhs", res.FString.Vars[0].Var)
		assert.Equal(t, " suffix", res.FString.Suffix)
	})

	t.Run("lhs fstring, rhs string", func(t *testing.T) {
		lhs := &ValueExpression{
			FString: &FString{
				Vars: []struct {
					Prefix string
					Var    string
					Config string
				}{
					{
						Prefix: "this is the lhs: ",
						Var:    "lhs",
						Config: "",
					},
				},
				Suffix: " suffix",
			},
		}

		rhs := &ValueExpression{
			String: "\" this is the right hand side\"",
		}

		res := concatStrings(lhs, rhs)

		require.NotNil(t, res.FString)
		assert.Len(t, res.FString.Vars, 1)
		assert.Equal(t, "this is the lhs: ", res.FString.Vars[0].Prefix)
		assert.Equal(t, "lhs", res.FString.Vars[0].Var)
		assert.Equal(t, " suffix this is the right hand side", res.FString.Suffix)
	})

	t.Run("both fstring", func(t *testing.T) {
		lhs := &ValueExpression{
			FString: &FString{
				Vars: []struct {
					Prefix string
					Var    string
					Config string
				}{
					{
						Prefix: "this is the lhs: ",
						Var:    "lhs",
						Config: "",
					},
				},
				Suffix: "lhs suffix",
			},
		}

		rhs := &ValueExpression{
			FString: &FString{
				Vars: []struct {
					Prefix string
					Var    string
					Config string
				}{
					{
						Prefix: " this is the rhs: ",
						Var:    "rhs",
						Config: "",
					},
				},
				Suffix: " rhs suffix",
			},
		}
		res := concatStrings(lhs, rhs)

		require.NotNil(t, res.FString)
		assert.Len(t, res.FString.Vars, 2)
		assert.Equal(t, "this is the lhs: ", res.FString.Vars[0].Prefix)
		assert.Equal(t, "lhs", res.FString.Vars[0].Var)

		assert.Equal(t, "lhs suffix this is the rhs: ", res.FString.Vars[1].Prefix)
		assert.Equal(t, "rhs", res.FString.Vars[1].Var)

		assert.Equal(t, " rhs suffix", res.FString.Suffix)
	})

	t.Run("both strings", func(t *testing.T) {
		lhs := &ValueExpression{
			String: "\"this is the left hand side\"",
		}

		rhs := &ValueExpression{
			String: "\" this is the right hand side\"",
		}
		res := concatStrings(lhs, rhs)

		require.NotEmpty(t, res.String)
		assert.Equal(t, "\"this is the left hand side this is the right hand side\"", res.String)
	})
}

func TestFStringImplicitStringConcat(t *testing.T) {
	str := "str('testing that we can carry these ' f'over {multiple} lines' r' \\n')"
	prog, err := newParser().parseAndHandleErrors(strings.NewReader(strings.ReplaceAll(str, "\t", "")))
	require.NoError(t, err)

	fString := prog[0].Ident.Action.Call.Arguments[0].Value.Val.FString
	assert.Equal(t, "testing that we can carry these over ", fString.Vars[0].Prefix)
	assert.Equal(t, "multiple", fString.Vars[0].Var)
	assert.Equal(t, " lines \\n", fString.Suffix)
}

// F strings should report a sensible error when the {} aren't complete
func TestFStringIncompleteError(t *testing.T) {
	str := "s = f'some {' '.join([])}'"
	_, err := newParser().parseAndHandleErrors(strings.NewReader(str))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unterminated brace in fstring")
}
