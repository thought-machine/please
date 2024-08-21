// This is essentially an end-to-end test on the whole thing; since it's
// quite tedious to write out the AST by hand we interpret sample BUILD files directly.

package asp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/src/core"
)

func parseFileToStatements(filename string) (*scope, []*Statement, error) {
	return parseFileToStatementsInPkg(filename, core.NewPackage("test/package"))
}

func parseFileToStatementsInPkg(filename string, pkg *core.Package) (*scope, []*Statement, error) {
	state := core.NewDefaultBuildState()
	state.Config.BuildConfig = map[string]string{"parser-engine": "python27"}
	parser := NewParser(state)

	src, err := rules.ReadAsset("builtins.build_defs")
	if err != nil {
		panic(err)
	}
	parser.MustLoadBuiltins("builtins.build_defs", src)
	statements, err := parser.parse(nil, filename)
	if err != nil {
		panic(err)
	}
	statements = parser.optimise(statements)
	parser.interpreter.optimiseExpressions(statements)
	s, err := parser.interpreter.interpretAll(pkg, nil, nil, 0, statements)
	return s, statements, err
}

func parseFile(filename string) (*scope, error) {
	s, _, err := parseFileToStatements(filename)
	return s, err
}

func TestInterpreterBasic(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/basic.build")
	require.NoError(t, err)
	assert.NotNil(t, s.Lookup("test"))
	assert.Panics(t, func() { s.Lookup("wibble") })
	assert.NotNil(t, s.Lookup("True"))
	assert.NotNil(t, s.Lookup("False"))
	assert.NotNil(t, s.Lookup("None"))
}

func TestInterpreterFunctionDef(t *testing.T) {
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

func TestInterpreterInterpreterOperators(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/operators.build")
	require.NoError(t, err)
	require.NotNil(t, s.Lookup("y"))
	i := s.Lookup("y").(pyInt)
	assert.EqualValues(t, 7, i)
	assert.True(t, s.Lookup("z").IsTruthy())
}

// Test that local is forced to be True if there are any system_srcs set
func TestSetLocalTrueIfSystemSrcs(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/system_srcs.build")
	require.NoError(t, err)
	assert.Equal(t, 2, s.pkg.NumTargets())
	assert.Equal(t, s.pkg.Target("system_srcs_set").Local, true)
	assert.Equal(t, s.pkg.Target("system_srcs_unset").Local, false)
}

func TestInterpreterInterpolation(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/interpolation.build")
	require.NoError(t, err)
	assert.EqualValues(t, "//abc:123", s.Lookup("x"))
}

func TestInterpreterCollections(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/collections.build")
	require.NoError(t, err)
	assert.EqualValues(t, True, s.Lookup("x"))
	assert.EqualValues(t, True, s.Lookup("y"))
	assert.EqualValues(t, False, s.Lookup("z"))
}

func TestInterpreterArguments(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/arguments.build")
	require.NoError(t, err)
	assert.EqualValues(t, "a:b:True", s.Lookup("x"))
	assert.EqualValues(t, "a:b:c", s.Lookup("y"))
	assert.EqualValues(t, "a:b:c", s.Lookup("z"))
}

func TestInterpreterMutableArguments(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/mutable_arguments.build")
	require.NoError(t, err)
	assert.EqualValues(t, 8, s.Lookup("y"))
}

func TestInterpreterBuiltins(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/builtins.build")
	require.NoError(t, err)
	assert.Equal(t, 1, s.pkg.NumTargets())
	assert.NotNil(t, s.pkg.Target("lib"))
}

func TestInterpreterConfig(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/config.build")
	require.NoError(t, err)
	for _, v := range []string{"g", "k1", "k2", "v", "i"} {
		assert.EqualValues(t, True, s.Lookup(v))
	}
}

func TestInterpreterParentheses(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/parentheses.build")
	require.NoError(t, err)
	assert.EqualValues(t, 1, s.Lookup("x"))
}

func TestInterpreterComprehensions(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/comprehensions.build")
	require.NoError(t, err)
	assert.EqualValues(t, pyList{pyString("file1"), pyString("file2")}, s.Lookup("file_srcs"))
	assert.EqualValues(t, pyList{pyString("file1+file1"), pyString("file1+file2"), pyString("file1+:rule1"),
		pyString("file2+file1"), pyString("file2+file2"), pyString("file2+:rule1")}, s.Lookup("pairs"))
}

func TestInterpreterEquality(t *testing.T) {
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

func TestInterpreterSlicing(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/slicing.build")
	require.NoError(t, err)
	assert.Equal(t, pyInt(2), s.Lookup("a"))
	assert.Equal(t, pyList{pyInt(2), pyInt(3)}, s.Lookup("b"))
	assert.Equal(t, pyList{pyInt(1)}, s.Lookup("c"))
	assert.Equal(t, pyList{pyInt(2)}, s.Lookup("d"))
	assert.Equal(t, pyInt(3), s.Lookup("e"))
	assert.Equal(t, pyList{pyInt(1), pyInt(2)}, s.Lookup("f"))
	assert.Equal(t, pyList{pyInt(1), pyInt(2), pyInt(3)}, s.Lookup("g"))
}

func TestInterpreterSorting(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/sorted.build")
	require.NoError(t, err)
	assert.Equal(t, pyList{pyInt(1), pyInt(2), pyInt(3)}, s.Lookup("y"))
	// N.B. sorted() sorts in-place, unlike Python's one. We may change that later.
}

func TestReversed(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/reversed.build")
	require.NoError(t, err)
	assert.Equal(t, pyList{}, s.Lookup("r1"))
	assert.Equal(t, pyList{pyInt(3), pyInt(2), pyInt(1)}, s.Lookup("r2"))
	assert.Equal(t, pyList{pyInt(4), pyInt(3), pyInt(2), pyInt(1)}, s.Lookup("r3"))
}

func TestFilter(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/filter.build")
	require.NoError(t, err)
	assert.Equal(t, pyList{pyInt(1), pyInt(2), pyInt(3)}, s.Lookup("f1"))
	assert.Equal(t, pyList{pyInt(0), pyInt(2)}, s.Lookup("f2"))
	assert.Equal(t, pyList{pyInt(5), pyInt(5)}, s.Lookup("f3"))
}

func TestMap(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/map.build")
	require.NoError(t, err)
	assert.Equal(t, pyList{pyInt(0), pyInt(1), pyInt(2), pyInt(3)}, s.Lookup("m1"))
	assert.Equal(t, pyList{pyInt(1), pyInt(2), pyInt(3), pyInt(4)}, s.Lookup("m2"))
}

func TestReduce(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/reduce.build")
	require.NoError(t, err)
	assert.Equal(t, pyInt(6), s.Lookup("r1"))
	assert.Equal(t, pyInt(16), s.Lookup("r2"))
	res := pyDict{
		"a": pyInt(2),
		"b": pyInt(3),
		"c": pyInt(4),
		"d": pyInt(5),
		"e": pyInt(0),
	}
	assert.Equal(t, res, s.Lookup("r3"))
	assert.Equal(t, s.Lookup("None"), s.Lookup("r4"))
	assert.Equal(t, pyInt(5), s.Lookup("r5"))
	assert.Equal(t, pyInt(6), s.Lookup("r6"))
	assert.Equal(t, pyInt(7), s.Lookup("r7"))
	assert.EqualValues(t, "abcde", s.Lookup("r8"))
}

func TestInterpreterUnpacking(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/unpacking.build")
	require.NoError(t, err)
	assert.EqualValues(t, "a", s.Lookup("a"))
	assert.EqualValues(t, "b", s.Lookup("b"))
	assert.EqualValues(t, "c", s.Lookup("c"))
	assert.EqualValues(t, "abc", s.Lookup("d"))
	assert.EqualValues(t, ".", s.Lookup("e"))
	assert.EqualValues(t, "def", s.Lookup("f"))
}

func TestInterpreterPrecedence(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/precedence.build")
	require.NoError(t, err)
	assert.EqualValues(t, pyList{pyString("a.go")}, s.Lookup("file_srcs"))
}

func TestInterpreterPrecedence2(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/precedence2.build")
	require.NoError(t, err)
	assert.True(t, s.Lookup("y").IsTruthy())
}

func TestInterpreterZip(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/zip.build")
	require.NoError(t, err)
	expected := pyList{
		pyList{pyInt(1), pyInt(4), pyInt(7)},
		pyList{pyInt(2), pyInt(5), pyInt(8)},
		pyList{pyInt(3), pyInt(6), pyInt(9)},
	}
	assert.EqualValues(t, expected, s.Lookup("x"))
}

func TestInterpreterOptimisations(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/optimisations.build")
	require.NoError(t, err)
	assert.EqualValues(t, "python2.7", s.Lookup("PARSER_LIB_NAME"))
}

func TestInterpreterContinue(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/continue.build")
	require.NoError(t, err)
	assert.EqualValues(t, pyList{pyInt(4), pyInt(5)}, s.Lookup("a"))
}

func TestInterpreterAliases(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/aliases.build")
	require.NoError(t, err)
	assert.EqualValues(t, 42, s.Lookup("v"))
}

func TestInterpreterPaths(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/paths.build")
	require.NoError(t, err)
	assert.EqualValues(t, "a/b/c", s.Lookup("a"))
	assert.EqualValues(t, "a/c", s.Lookup("b"))
	assert.EqualValues(t, pyList{pyString("a/b"), pyString("c")}, s.Lookup("c"))
	assert.EqualValues(t, pyList{pyString(""), pyString("a")}, s.Lookup("d"))
	assert.EqualValues(t, pyList{pyString("a/test"), pyString(".txt")}, s.Lookup("e"))
	assert.EqualValues(t, pyList{pyString("a/test"), pyString("")}, s.Lookup("f"))
	assert.EqualValues(t, "c", s.Lookup("g"))
	assert.EqualValues(t, "a", s.Lookup("h"))
	assert.EqualValues(t, "a/b", s.Lookup("i"))
	assert.EqualValues(t, "", s.Lookup("j"))
}

func TestInterpreterStrings(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/strings.build")
	require.NoError(t, err)
	assert.EqualValues(t, pyList{
		pyString("acpi"), pyString("base64"), pyString("basename"), pyString("blkid"), pyString("blockdev"),
		pyString("bunzip2"), pyString("bzcat"), pyString("cal"), pyString("cat"), pyString("catv"),
		pyString("chattr"), pyString("whoami"), pyString("xargs"), pyString("xxd"), pyString("yes"),
	}, s.Lookup("TOYS2"))
	assert.EqualValues(t, "acpi base64 basename blkid blockdev bunzip2 bzcat cal cat catv chattr\nwhoami xargs xxd yes", s.Lookup("TOYS3"))
}

func TestInterpreterArgumentCompatibility(t *testing.T) {
	// This isn't a totally obvious property of the interpreter, but when an argument specifies
	// a type and is given None, we adopt the default. This allows external functions to use None
	// for various arguments (e.g. deps), but internally we can treat them as lists.
	s, err := parseFile("src/parse/asp/test_data/interpreter/argument_compatibility.build")
	require.NoError(t, err)
	assert.EqualValues(t, pyList{pyInt(1)}, s.Lookup("x"))
}

func TestInterpreterOptimiseConfig(t *testing.T) {
	s, statements, err := parseFileToStatements("src/parse/asp/test_data/interpreter/optimise_config.build")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statements))
	assert.NotNil(t, statements[0].Ident)
	assert.NotNil(t, statements[0].Ident.Action)
	assert.NotNil(t, statements[0].Ident.Action.Assign)
	assert.Equal(t, "GO_TOOL", statements[0].Ident.Action.Assign.optimised.Config)
	assert.EqualValues(t, "go", s.Lookup("x"))
}

func TestInterpreterOptimiseJoin(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/optimise_join.build")
	require.NoError(t, err)
	assert.EqualValues(t, "1 2 3", s.Lookup("x"))
	assert.EqualValues(t, "1 2 3", s.Lookup("y"))
}

func TestInterpreterPartition(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/partition.build")
	assert.NoError(t, err)
	assert.EqualValues(t, "27", s.Lookup("major"))
	assert.EqualValues(t, ".0.", s.Lookup("mid"))
	assert.EqualValues(t, "3", s.Lookup("minor"))
	assert.EqualValues(t, "begin ", s.Lookup("start"))
	assert.EqualValues(t, "sep", s.Lookup("sep"))
	assert.EqualValues(t, " end", s.Lookup("end"))
}

func TestInterpreterFStrings(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/fstring.build")
	assert.NoError(t, err)
	assert.EqualValues(t, "a", s.Lookup("x"))
	assert.EqualValues(t, "a", s.Lookup("y"))
	assert.EqualValues(t, "x: a y: a fin", s.Lookup("z"))
	assert.EqualValues(t, "6", s.Lookup("nest_test"))
}

func TestInterpreterSubincludeConfig(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/partition.build")
	assert.NoError(t, err)
	s.SetAll(s.interpreter.Subinclude(s, "src/parse/asp/test_data/interpreter/subinclude_config.build", core.NewPackage("test").Label(), false), false)
	assert.EqualValues(t, "test test", s.config.Get("test", None))
}

func TestInterpreterValidateReturnVal(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/return_type.build")
	assert.NotNil(t, s.Lookup("subinclude"))
	assert.Error(t, err, "Invalid return type str from function dict_val, expecting dict")
}

func TestInterpreterLen(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/len.build")
	assert.NoError(t, err)
	assert.EqualValues(t, "sync", s.Lookup("y"))
	assert.EqualValues(t, 4, s.Lookup("l1"))
	assert.EqualValues(t, 6, s.Lookup("l2"))
	assert.EqualValues(t, 5, s.Lookup("l3"))
	assert.EqualValues(t, 2, s.Lookup("l4"))
}

func TestInterpreterIndex(t *testing.T) {
	t.Run("String indexing", func(t *testing.T) {
		s, err := parseFile("src/parse/asp/test_data/interpreter/index_string.build")
		assert.NoError(t, err)
		assert.EqualValues(t, pyString("l"), s.Lookup("c1"))
		assert.EqualValues(t, pyString("n"), s.Lookup("c2"))
		assert.EqualValues(t, pyString("\u043d"), s.Lookup("c3"))
		assert.EqualValues(t, pyString("\u0637"), s.Lookup("c4"))
	})
}

func TestInterpreterFStringDollars(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/fstrings.build")
	assert.NoError(t, err)
	assert.EqualValues(t, "mickey donald ${goofy} {sora}", s.Lookup("z"))
}

func TestInterpreterDoubleIndex(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/double_index.build")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, s.Lookup("y"))
}

func TestInterpreterSubincludeAll(t *testing.T) {
	_, err := parseFile("src/parse/asp/test_data/interpreter/subinclude_all.build")
	assert.Error(t, err)
}

func TestInterpreterDictUnion(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/dict_union.build")
	assert.NoError(t, err)
	assert.EqualValues(t, pyDict{
		"mickey": pyInt(1),
		"donald": pyInt(2),
		"goofy":  pyInt(3),
	}, s.Lookup("z"))
}

func TestIsNotNone(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/isnot.build")
	assert.NoError(t, err)
	assert.Equal(t, True, s.Lookup("x"))
}

func TestRemoveAffixes(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/remove_affixes.build")
	assert.NoError(t, err)
	assert.EqualValues(t, "PEP 616: New removeprefix() and removesuffix() string methods", s.Lookup("x"))
	assert.EqualValues(t, "New removeprefix() and removesuffix() string methods", s.Lookup("y"))
	assert.EqualValues(t, "removeprefix() and removesuffix() ", s.Lookup("z"))
}

func TestSubrepoName(t *testing.T) {
	pkg := core.NewPackage("test/pkg")
	pkg.SubrepoName = "pleasings"
	s, _, err := parseFileToStatementsInPkg("src/parse/asp/test_data/interpreter/subrepo_name.build", pkg)
	assert.NoError(t, err)

	assert.EqualValues(t, "pleasings", s.Lookup("subrepo"))
}

func TestMultiply(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/multiply.build")
	assert.NoError(t, err)
	assert.EqualValues(t, 42, s.Lookup("i1"))
	assert.EqualValues(t, 42, s.Lookup("i2"))
	assert.EqualValues(t, "abcabcabc", s.Lookup("s1"))
	assert.EqualValues(t, "abcabcabc", s.Lookup("s2"))
	assert.EqualValues(t, pyList{pyString("a"), pyString("b"), pyString("a"), pyString("b")}, s.Lookup("l1"))
	assert.EqualValues(t, pyList{pyString("a"), pyString("b"), pyString("a"), pyString("b")}, s.Lookup("l2"))
}

func TestDivide(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/divide.build")
	assert.NoError(t, err)
	assert.EqualValues(t, 0, s.Lookup("i"))
	assert.EqualValues(t, 7, s.Lookup("j"))
	assert.EqualValues(t, -2, s.Lookup("k"))
}

func TestFStringOptimisation(t *testing.T) {
	s, stmts, err := parseFileToStatements("src/parse/asp/test_data/interpreter/fstring_optimisation.build")
	require.NoError(t, err)
	assert.EqualValues(t, s.Lookup("x"), "test")
	// Check that it's been optimised to something
	assign := stmts[0].Ident.Action.Assign
	assert.Nil(t, assign.Val)
	assert.NotNil(t, assign.optimised.Constant)
	assert.EqualValues(t, "test", assign.optimised.Constant)
}

func TestFormat(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/format.build")
	assert.NoError(t, err)
	assert.EqualValues(t, `LLVM_NATIVE_ARCH=\"x86\"`, s.Lookup("arch"))
	assert.EqualValues(t, `ARCH="linux_amd64"`, s.Lookup("arch2"))
}

func TestAny(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s, err := parseFile("src/parse/asp/test_data/interpreter/any.build")
		assert.NoError(t, err)
		for i := 1; i <= 9; i++ {
			assert.EqualValues(t, pyBool(true), s.Lookup(fmt.Sprintf("t%d", i)))
		}
		for i := 1; i <= 9; i++ {
			assert.EqualValues(t, pyBool(false), s.Lookup(fmt.Sprintf("f%d", i)))
		}
	})
}

func TestAll(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s, err := parseFile("src/parse/asp/test_data/interpreter/all.build")
		assert.NoError(t, err)
		for i := 1; i <= 9; i++ {
			assert.EqualValues(t, pyBool(true), s.Lookup(fmt.Sprintf("t%d", i)))
		}
		for i := 1; i <= 9; i++ {
			assert.EqualValues(t, pyBool(false), s.Lookup(fmt.Sprintf("f%d", i)))
		}
	})
}

func TestMin(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s, err := parseFile("src/parse/asp/test_data/interpreter/min.build")
		assert.NoError(t, err)
		for i := 1; i <= 3; i++ {
			assert.EqualValues(t, pyInt(1), s.Lookup(fmt.Sprintf("i%d", i)))
			assert.EqualValues(t, pyString("five"), s.Lookup(fmt.Sprintf("s%d", i)))
		}
		for i := 4; i <= 6; i++ {
			assert.EqualValues(t, pyString("ten"), s.Lookup(fmt.Sprintf("s%d", i)))
		}
		assert.EqualValues(t, pyString("one"), s.Lookup("s7"))
	})
}

func TestMax(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s, err := parseFile("src/parse/asp/test_data/interpreter/max.build")
		assert.NoError(t, err)
		for i := 1; i <= 3; i++ {
			assert.EqualValues(t, pyInt(5), s.Lookup(fmt.Sprintf("i%d", i)))
			assert.EqualValues(t, pyString("two"), s.Lookup(fmt.Sprintf("s%d", i)))
		}
		for i := 4; i <= 6; i++ {
			assert.EqualValues(t, pyString("three"), s.Lookup(fmt.Sprintf("s%d", i)))
		}
		assert.EqualValues(t, pyString("one"), s.Lookup("s7"))
	})
}

func TestChr(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s, err := parseFile("src/parse/asp/test_data/interpreter/chr.build")
		assert.NoError(t, err)
		assert.EqualValues(t, pyString("\x00"), s.Lookup("null"))
		assert.EqualValues(t, pyString("a"), s.Lookup("a"))
		assert.EqualValues(t, pyString("€"), s.Lookup("euro"))
		assert.EqualValues(t, pyString("\U0010FFFF"), s.Lookup("maximum"))
	})
	t.Run("Wrong parameter type", func(t *testing.T) {
		_, err := parseFile("src/parse/asp/test_data/interpreter/chr_wrong_type.build")
		assert.ErrorContains(t, err, "Invalid type for argument i to chr; expected int, was str")
	})
	t.Run("Parameter out of bounds (too low)", func(t *testing.T) {
		_, err := parseFile("src/parse/asp/test_data/interpreter/chr_bounds_low.build")
		assert.ErrorContains(t, err, "Argument i must be within the Unicode code point range")
	})
	t.Run("Parameter out of bounds (too high)", func(t *testing.T) {
		_, err := parseFile("src/parse/asp/test_data/interpreter/chr_bounds_high.build")
		assert.ErrorContains(t, err, "Argument i must be within the Unicode code point range")
	})
}

func TestOrd(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s, err := parseFile("src/parse/asp/test_data/interpreter/ord.build")
		assert.NoError(t, err)
		assert.EqualValues(t, pyInt(97), s.Lookup("a"))
		assert.EqualValues(t, pyInt(8364), s.Lookup("euro"))
		assert.EqualValues(t, pyInt(8984), s.Lookup("cmd"))
	})
	t.Run("Wrong parameter type", func(t *testing.T) {
		_, err := parseFile("src/parse/asp/test_data/interpreter/ord_wrong_type.build")
		assert.ErrorContains(t, err, "Invalid type for argument c to ord; expected str, was int")
	})
	t.Run("Parameter too short", func(t *testing.T) {
		_, err := parseFile("src/parse/asp/test_data/interpreter/ord_empty.build")
		assert.ErrorContains(t, err, "Argument c must be a string containing a single Unicode character")
	})
	t.Run("Parameter out of bounds (too high)", func(t *testing.T) {
		_, err := parseFile("src/parse/asp/test_data/interpreter/ord_multiple.build")
		assert.ErrorContains(t, err, "Argument c must be a string containing a single Unicode character")
	})
}

func TestIsSemver(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s, err := parseFile("src/parse/asp/test_data/interpreter/is_semver.build")
		assert.NoError(t, err)
		for i := 1; i <= 18; i++ {
			assert.EqualValues(t, pyBool(true), s.Lookup(fmt.Sprintf("t%d", i)))
		}
		for i := 1; i <= 16; i++ {
			assert.EqualValues(t, pyBool(false), s.Lookup(fmt.Sprintf("f%d", i)))
		}
	})
}

func TestJSON(t *testing.T) {
	state := core.NewDefaultBuildState()
	parser := NewParser(state)

	src, err := rules.ReadAsset("builtins.build_defs")
	if err != nil {
		panic(err)
	}
	parser.MustLoadBuiltins("builtins.build_defs", src)
	statements, err := parser.parse(nil, "src/parse/asp/test_data/json.build")
	if err != nil {
		panic(err)
	}
	statements = parser.optimise(statements)
	parser.interpreter.optimiseExpressions(statements)

	s := parser.interpreter.scope.NewScope("BUILD", core.ParseModeNormal)

	list := pyList{pyString("foo"), pyInt(5)}
	dict := pyDict{"foo": pyString("bar")}
	confBase := &pyConfigBase{dict: dict}
	config := &pyConfig{base: confBase, overlay: pyDict{"baz": pyInt(6)}}

	s.locals["some_list"] = list
	s.locals["some_frozen_list"] = list.Freeze()
	s.locals["some_dict"] = dict
	s.locals["some_frozen_dict"] = dict.Freeze()
	s.locals["some_config"] = config
	s.locals["some_frozen_config"] = config.Freeze()

	s.interpretStatements(statements)

	assert.Equal(t, "[\"foo\",5]", s.Lookup("json_list").String())
	assert.Equal(t, "[\"foo\",5]", s.Lookup("json_frozen_list").String())
	assert.Equal(t, "{\"foo\":\"bar\"}", s.Lookup("json_dict").String())
	assert.Equal(t, "{\"foo\":\"bar\"}", s.Lookup("json_frozen_dict").String())
	assert.Contains(t, s.Lookup("json_config").String(), "\"foo\":\"bar\"")
	assert.Contains(t, s.Lookup("json_config").String(), "\"baz\":6")
	assert.Contains(t, s.Lookup("json_frozen_config").String(), "\"foo\":\"bar\"")
	assert.Contains(t, s.Lookup("json_frozen_config").String(), "\"baz\":6")
	assert.EqualValues(t, "[0,1,2,3]", s.Lookup("json_range"))
	assert.Equal(t, "{\n  \"foo\": \"bar\"\n}", s.Lookup("json_pretty").String())
}

func TestSemverCheck(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s, err := parseFile("src/parse/asp/test_data/interpreter/semver_check.build")
		assert.NoError(t, err)
		assert.EqualValues(t, pyBool(true), s.Lookup("c1"))
		assert.EqualValues(t, pyBool(false), s.Lookup("c2"))
		assert.EqualValues(t, pyBool(true), s.Lookup("c3"))
		assert.EqualValues(t, pyBool(true), s.Lookup("c4"))
	})

	t.Run("InvalidVersion", func(t *testing.T) {
		_, err := parseFile("src/parse/asp/test_data/interpreter/semver_check_invalid_version.build")
		assert.Error(t, err)
	})

	t.Run("InvalidConstraint", func(t *testing.T) {
		_, err := parseFile("src/parse/asp/test_data/interpreter/semver_check_invalid_constraint.build")
		assert.Error(t, err)
	})
}

func TestLogConfigVariable(t *testing.T) {
	state := core.NewDefaultBuildState()
	parser := NewParser(state)

	src, err := rules.ReadAsset("builtins.build_defs")
	if err != nil {
		panic(err)
	}
	parser.MustLoadBuiltins("builtins.build_defs", src)
	statements, err := parser.parse(nil, "src/parse/asp/test_data/log_config.build")
	if err != nil {
		panic(err)
	}
	statements = parser.optimise(statements)
	parser.interpreter.optimiseExpressions(statements)

	list := pyList{pyString("foo"), pyInt(5)}
	dict := pyDict{"foo": pyString("bar"), "baz": list}
	confBase := &pyConfigBase{dict: dict}
	config := &pyConfig{base: confBase, overlay: pyDict{"baz": pyInt(6)}}

	s := parser.interpreter.scope.NewScope("BUILD", core.ParseModeNormal)
	s.config = config
	s.Set("CONFIG", config)

	capturedOutput := ""
	capture := func(format string, args ...interface{}) {
		capturedOutput = fmt.Sprintf(format, args...)
	}

	setLogCode(s, "info", capture)
	s.interpretStatements(statements)

	assert.Equal(t, `//: {"baz": 6, "foo": bar}`, capturedOutput)
}

func TestOperatorPrecedence(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/operator_precedence.build")
	assert.NoError(t, err)
	assert.EqualValues(t, 15, s.Lookup("a"))
	assert.EqualValues(t, 53, s.Lookup("b"))
	assert.EqualValues(t, 32, s.Lookup("c"))
	assert.EqualValues(t, False, s.Lookup("d"))
	assert.EqualValues(t, False, s.Lookup("e"))
	assert.EqualValues(t, True, s.Lookup("f"))
	assert.EqualValues(t, 5, s.Lookup("g"))
	assert.EqualValues(t, 2, s.Lookup("h"))
	assert.EqualValues(t, "bc", s.Lookup("i"))
	assert.EqualValues(t, "a", s.Lookup("j"))
	assert.EqualValues(t, 1, s.Lookup("k"))
	assert.EqualValues(t, 1, s.Lookup("l"))
	assert.EqualValues(t, False, s.Lookup("m"))
	assert.EqualValues(t, True, s.Lookup("n"))
}

func TestListConcatenation(t *testing.T) {
	s, err := parseFile("src/parse/asp/test_data/interpreter/list_concat.build")
	assert.NoError(t, err)
	assert.EqualValues(t, pyList{
		pyString("apple"),
		pyString("banana"),
		pyString("edamame"),
		pyString("fennel"),
		pyString("tuna"),
		pyString("baked beans"),
		pyString("haribo"),
	}, s.Lookup("fruit_veg_canned_food_and_sweets"))
}
