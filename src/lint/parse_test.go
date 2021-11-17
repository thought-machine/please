package lint

import (
	"os"
	"testing"

	"github.com/peterebden/go-deferred-regex"
	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestParseLintLine(t *testing.T) {
	const in = "src/core/lint.go:21:2  structcheck  `y` is unused\n"
	linter := &core.Linter{
		OutputFormat: deferredregex.DeferredRegex{
			Re: "(?P<file>[^:]+):(?P<line>[0-9]+):(?P<col>[0-9]+) +(?P<linter>[a-z]+) +(?P<message>.*)$",
		},
	}
	assert.Equal(t, []core.LintResult{{
		Linter:  "structcheck", // This should override 'golangci-lint' that we passed in
		File:    "src/core/lint.go",
		Line:    21,
		Col:     2,
		Message: "`y` is unused",
	}}, parseLintLines(linter, "golangci-lint", in))
}

func TestParseRewrites(t *testing.T) {
	mustReadFile := func(filename string) string {
		b, err := os.ReadFile(filename)
		assert.NoError(t, err)
		return string(b)
	}

	before := mustReadFile("src/lint/test_data/test.go")
	after := mustReadFile("src/lint/test_data/rewritten.go")
	patch := mustReadFile("src/lint/test_data/patch.diff")

	assert.Equal(t, []core.LintResult{
		{
			Linter:   "gofmt",
			File:     "src/lint/test_data/test.go",
			Line:     2,
			Patch:    patch,
			Severity: "autoformat",
		},
	}, computeDiffs("gofmt", "src/lint/test_data/test.go", before, after))
}
