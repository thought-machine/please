// This is essentially an end-to-end test on the whole thing; since it's
// quite tedious to write out the AST by hand we interpret sample BUILD files directly.

package asp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"core"
	"parse/asp/builtins"
)

func parseFile(filename string) (*scope, error) {
	state := core.NewBuildState(1, nil, 4, core.DefaultConfiguration())
	state.Config.BuildConfig = map[string]string{"parser-engine": "python27"}
	pkg := core.NewPackage("test/package")
	parser := NewParser(state)
	parser.MustLoadBuiltins("builtins.build_defs", nil, builtins.MustAsset("builtins.build_defs.gob"))
	statements, err := parser.parse(filename)
	if err != nil {
		panic(err)
	}
	statements = parser.optimise(statements)
	parser.interpreter.optimiseExpressions(reflect.ValueOf(statements))
	return parser.interpreter.interpretAll(pkg, statements)
}

func TestBasic(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/basic.build")
	require.NoError(t, err)
	assert.NotNil(t, s.Lookup("test"))
	assert.Panics(t, func() { s.Lookup("wibble") })
	assert.NotNil(t, s.Lookup("True"))
	assert.NotNil(t, s.Lookup("False"))
	assert.NotNil(t, s.Lookup("None"))
}

func TestFunctionDef(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/function_def.build")
	require.NoError(t, err)
	require.NotNil(t, s.Lookup("cc_library"))
	f := s.Lookup("cc_library").(*pyFunc)
	assert.Equal(t, 14, len(f.args))
	assert.Equal(t, 14, len(f.constants))
	assert.Equal(t, 0, len(f.defaults))
	assert.Equal(t, "name", f.args[0])
	assert.Nil(t, f.constants[0])
	assert.Equal(t, "srcs", f.args[1])
	assert.NotNil(t, f.constants[1])
}

func TestOperators(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/operators.build")
	require.NoError(t, err)
	require.NotNil(t, s.Lookup("y"))
	i := s.Lookup("y").(pyInt)
	assert.EqualValues(t, 7, i)
	assert.True(t, s.Lookup("z").IsTruthy())
}

func TestInterpolation(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/interpolation.build")
	require.NoError(t, err)
	assert.EqualValues(t, "//abc:123", s.Lookup("x"))
}

func TestCollections(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/collections.build")
	require.NoError(t, err)
	assert.EqualValues(t, True, s.Lookup("x"))
	assert.EqualValues(t, True, s.Lookup("y"))
	assert.EqualValues(t, False, s.Lookup("z"))
}

func TestArguments(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/arguments.build")
	require.NoError(t, err)
	assert.EqualValues(t, "a:b:True", s.Lookup("x"))
	assert.EqualValues(t, "a:b:c", s.Lookup("y"))
	assert.EqualValues(t, "a:b:c", s.Lookup("z"))
}

func TestMutableArguments(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/mutable_arguments.build")
	require.NoError(t, err)
	assert.EqualValues(t, 8, s.Lookup("y"))
}

func TestBuiltins(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/builtins.build")
	require.NoError(t, err)
	assert.Equal(t, 1, s.pkg.NumTargets())
	assert.NotNil(t, s.pkg.Target("lib"))
}

func TestParentheses(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/parentheses.build")
	require.NoError(t, err)
	assert.EqualValues(t, 1, s.Lookup("x"))
}

func TestComprehensions(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/comprehensions.build")
	require.NoError(t, err)
	assert.EqualValues(t, pyList{pyString("file1"), pyString("file2")}, s.Lookup("file_srcs"))
	assert.EqualValues(t, pyList{pyString("file1+file1"), pyString("file1+file2"), pyString("file1+:rule1"),
		pyString("file2+file1"), pyString("file2+file2"), pyString("file2+:rule1")}, s.Lookup("pairs"))
}

func TestEquality(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/equality.build")
	require.NoError(t, err)
	assert.Equal(t, True, s.Lookup("a"))
	assert.Equal(t, True, s.Lookup("b"))
	assert.Equal(t, False, s.Lookup("c"))
	assert.Equal(t, False, s.Lookup("d"))
	assert.Equal(t, True, s.Lookup("e"))
	assert.Equal(t, False, s.Lookup("f"))
	assert.Equal(t, False, s.Lookup("g"))
}

func TestSlicing(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/slicing.build")
	require.NoError(t, err)
	assert.Equal(t, pyInt(2), s.Lookup("a"))
	assert.Equal(t, pyList{pyInt(2), pyInt(3)}, s.Lookup("b"))
	assert.Equal(t, pyList{pyInt(1)}, s.Lookup("c"))
	assert.Equal(t, pyList{pyInt(2)}, s.Lookup("d"))
	assert.Equal(t, pyInt(3), s.Lookup("e"))
	assert.Equal(t, pyList{pyInt(1), pyInt(2)}, s.Lookup("f"))
}

func TestSorting(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/sorted.build")
	require.NoError(t, err)
	assert.Equal(t, pyList{pyInt(1), pyInt(2), pyInt(3)}, s.Lookup("y"))
	// N.B. sorted() sorts in-place, unlike Python's one. We may change that later.
}

func TestUnpacking(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/unpacking.build")
	require.NoError(t, err)
	assert.EqualValues(t, "a", s.Lookup("a"))
	assert.EqualValues(t, "b", s.Lookup("b"))
	assert.EqualValues(t, "c", s.Lookup("c"))
	assert.EqualValues(t, "abc", s.Lookup("d"))
	assert.EqualValues(t, ".", s.Lookup("e"))
	assert.EqualValues(t, "def", s.Lookup("f"))
}

func TestPrecedence(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/precedence.build")
	require.NoError(t, err)
	assert.EqualValues(t, pyList{pyString("a.go")}, s.Lookup("file_srcs"))
}

func TestZip(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/zip.build")
	require.NoError(t, err)
	expected := pyList{
		pyList{pyInt(1), pyInt(4), pyInt(7)},
		pyList{pyInt(2), pyInt(5), pyInt(8)},
		pyList{pyInt(3), pyInt(6), pyInt(9)},
	}
	assert.EqualValues(t, expected, s.Lookup("x"))
}

func TestOptimisations(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/optimisations.build")
	require.NoError(t, err)
	assert.EqualValues(t, "python2.7", s.Lookup("PARSER_LIB_NAME"))
}

func TestContinue(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/continue.build")
	require.NoError(t, err)
	assert.EqualValues(t, pyList{pyInt(4), pyInt(5)}, s.Lookup("a"))
}
