// Tests on the logging functionality. These are kept separate because we
// hijack the logger backend to test it which makes it hard to follow what's
// going on for other tests.

package asp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/src/core"
)

type record struct { //nolint:unused
	Level logging.Level
	Msg   string
}

func parseFile2(filename string) (*scope, error) {
	state := core.NewDefaultBuildState()
	pkg := core.NewPackage("test/package")
	pkg.Filename = "test/package/BUILD"
	parser := NewParser(state)
	src, err := rules.ReadAsset("builtins.build_defs")
	if err != nil {
		panic(err)
	}
	parser.MustLoadBuiltins("builtins.build_defs", src)
	statements, err := parser.parse(filename)
	if err != nil {
		panic(err)
	}
	return parser.interpreter.interpretAll(pkg, nil, nil, false, statements)
}

// assertRecords asserts equality of a series of logging records.
func assertRecords(t *testing.T, backend *logging.MemoryBackend, expected []record) {
	t.Helper()
	actual := []record{}
	for node := backend.Head(); node != nil; node = node.Next() {
		actual = append(actual, record{node.Record.Level, node.Record.Message()})
	}
	assert.Equal(t, expected, actual)
}

func TestLogNotice(t *testing.T) {
	backend := logging.InitForTesting(logging.NOTICE)
	_, err := parseFile2("src/parse/asp/test_data/interpreter/log.build")
	require.NoError(t, err)
	assertRecords(t, backend, []record{
		{logging.NOTICE, "//test/package/BUILD: notice"},
		{logging.WARNING, "//test/package/BUILD: warning"},
		{logging.ERROR, "//test/package/BUILD: error"},
	})
}

func TestLogInfo(t *testing.T) {
	// N.B. we don't test at DEBUG because then other things get logged from the parser.
	backend := logging.InitForTesting(logging.INFO)
	_, err := parseFile2("src/parse/asp/test_data/interpreter/log.build")
	require.NoError(t, err)
	assertRecords(t, backend, []record{
		{logging.INFO, "//test/package/BUILD: info"},
		{logging.NOTICE, "//test/package/BUILD: notice"},
		{logging.WARNING, "//test/package/BUILD: warning"},
		{logging.ERROR, "//test/package/BUILD: error"},
	})
}
