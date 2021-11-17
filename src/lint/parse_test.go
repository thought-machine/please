package lint

import (
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
