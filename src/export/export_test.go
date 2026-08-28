package export

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
)

func TestMinimalSubincludeStatement(t *testing.T) {
	testCases := []struct {
		name            string
		statementStr    string
		availableLabels []core.BuildLabel
		requiredLabels  []core.BuildLabel
		out             string
	}{
		{
			name:            "Successful no pruning subinclude",
			statementStr:    `subinclude("//build_defs:test")`,
			availableLabels: core.ParseBuildLabels([]string{"//build_defs:test"}),
			requiredLabels:  core.ParseBuildLabels([]string{"//build_defs:test"}),
			out:             `subinclude("//build_defs:test")`,
		},

		{
			name:            "Single subinclude (not required)",
			statementStr:    `subinclude("//build_defs:other")`,
			availableLabels: core.ParseBuildLabels([]string{"//build_defs:other"}),
			requiredLabels:  nil,
			out:             "",
		},
		{
			name:            "Multiple subincludes (sorted and filtered)",
			statementStr:    `subinclude("//build_defs:test", "//build_defs:abc", "//build_defs:other")`,
			availableLabels: core.ParseBuildLabels([]string{"//build_defs:test", "//build_defs:abc", "//build_defs:other"}),
			requiredLabels:  core.ParseBuildLabels([]string{"//build_defs:test", "//build_defs:abc"}),
			out: "subinclude(\n" +
				"    \"//build_defs:test\",\n" +
				"    \"//build_defs:abc\",\n" +
				")",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := asp.NewParserOnly()
			statements, err := p.ParseData([]byte(tc.statementStr), "BUILD")
			assert.NoError(t, err)
			assert.Len(t, statements, 1)
			stmt := statements[0]

			e := newExporter(nil, "", false).impl.(*trimmedExporter)

			pkg := &core.Package{Name: "test"}
			e.requiredSubincludes[pkg.Label()] = tc.requiredLabels
			trimmer := trimmer{
				pkg:      pkg,
				exporter: e,
				origin:   []byte(tc.statementStr),
			}

			assert.Equal(t, tc.out, trimmer.minimalSubincludeStatement(stmt, tc.availableLabels))
		})
	}
}

func TestFilterPackageFile(t *testing.T) {
	testCases := []struct {
		name     string
		required []string
		expected string
	}{
		{
			name:     "Keep only A",
			required: []string{"a"},
			expected: "src/export/test_data/filter_expected_a.build",
		},
		{
			name:     "Keep only B",
			required: []string{"b"},
			expected: "src/export/test_data/filter_expected_b.build",
		},
		{
			name:     "Keep both",
			required: []string{"a", "b"},
			expected: "src/export/test_data/filter.build",
		},
		{
			name:     "Keep none",
			required: []string{},
			expected: "src/export/test_data/filter_expected_none.build",
		},
	}

	contentPath := "src/export/test_data/filter.build"

	p := asp.NewParserOnly()
	statements, err := p.ParseFileOnly(contentPath)
	assert.NoError(t, err)

	pkg := core.NewPackage("test", core.WithPackageMetadata())
	pkg.Filename = contentPath
	targetLabels := walkASTRegisterTargets(t, statements, pkg, nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := newExporter(nil, "", false).impl.(*trimmedExporter)
			for _, name := range tc.required {
				e.exportedTargets[targetLabels[name]] = true
			}
			e.visitedPackages[pkg.Label()] = true

			p := asp.NewParserOnly()
			got, err := e.trimPackage(p, pkg)
			got = trimNewlines(got)
			assert.NoError(t, err)

			expected, err := os.ReadFile(tc.expected)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), string(got))
		})
	}
}

// walkASTRegisterTargets is a test helper to register simple targets and their build statements.
func walkASTRegisterTargets(t *testing.T, stmts []*asp.Statement, pkg *core.Package, toRegister []string) map[string]core.BuildLabel {
	t.Helper()
	targetLabels := map[string]core.BuildLabel{}
	asp.WalkAST(stmts, func(stmt *asp.Statement) bool {
		arg := asp.FindArgument(stmt, "name")
		if arg == nil {
			return true // Continue
		}

		// Not in targets we want to register, continue. Empty selection
		// will cause all targets to be registered.
		name := strings.Trim(arg.Value.Val.String, "\"")
		if toRegister != nil && !slices.Contains(toRegister, name) {
			return true
		}

		label := core.NewBuildLabel(pkg.Name, name)
		targetLabels[name] = label
		target := &core.BuildTarget{Label: label}
		pkg.Metadata.RegisterTargetStatement(target.Label, func() core.BuildStatement {
			return asp.NewBuildStatement(stmt)
		})
		return true
	})
	return targetLabels
}
