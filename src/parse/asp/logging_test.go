// Tests on the logging functionality. These are kept separate because we
// hijack the logger backend to test it which makes it hard to follow what's
// going on for other tests.

package asp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/rules"
)

type record struct {
	Level logging.Level
	Msg   string
}

func parseFile(filename string) (*scope, error) {
	state := core.NewBuildState(1, nil, 4, core.DefaultConfiguration())
	pkg := core.NewPackage("test/package")
	pkg.Filename = "test/package/BUILD"
	parser := NewParser(state)
	parser.MustLoadBuiltins("builtins.build_defs", nil, rules.MustAsset("builtins.build_defs.gob"))
	statements, err := parser.parse(filename)
	if err != nil {
		panic(err)
	}
	return parser.interpreter.interpretAll(pkg, statements)
}

// assertRecords asserts equality of a series of logging records.
func assertRecords(t *testing.T, backend *logging.MemoryBackend, expected []record) {
	actual := []record{}
	for node := backend.Head(); node != nil; node = node.Next() {
		actual = append(actual, record{node.Record.Level, node.Record.Message()})
	}
	assert.Equal(t, expected, actual)
}

func TestLogNotice(t *testing.T) {
	backend := logging.InitForTesting(logging.NOTICE)
	_, err := parseFile("src/parse/asp/test_data/interpreter/log.build")
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
	_, err := parseFile("src/parse/asp/test_data/interpreter/log.build")
	require.NoError(t, err)
	assertRecords(t, backend, []record{
		{logging.INFO, "//test/package/BUILD: info"},
		{logging.NOTICE, "//test/package/BUILD: notice"},
		{logging.WARNING, "//test/package/BUILD: warning"},
		{logging.ERROR, "//test/package/BUILD: error"},
	})
}
