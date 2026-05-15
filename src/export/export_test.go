package export

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
)

func TestMinimalSubincludeStatement(t *testing.T) {
	testCases := []struct {
		name            string
		availableLabels []core.BuildLabel
		requiredLabels  []core.BuildLabel
		out             string
	}{
		{
			name:            "Successful no pruning subinclude",
			availableLabels: core.ParseBuildLabels([]string{"//build_defs:test"}),
			requiredLabels:  core.ParseBuildLabels([]string{"//build_defs:test"}),
			out:             `subinclude("//build_defs:test")`,
		},
		{
			name:            "No subincludes",
			availableLabels: nil,
			requiredLabels:  nil,
			out:             "",
		},
		{
			name:            "Single subinclude (not required)",
			availableLabels: core.ParseBuildLabels([]string{"//build_defs:other"}),
			requiredLabels:  nil,
			out:             "",
		},
		{
			name:            "Multiple subincludes (sorted and filtered)",
			availableLabels: core.ParseBuildLabels([]string{"//build_defs:test", "//build_defs:abc", "//build_defs:other"}),
			requiredLabels:  core.ParseBuildLabels([]string{"//build_defs:test", "//build_defs:abc"}),
			out: "subinclude(\n" +
				"    \"//build_defs:abc\",\n" +
				"    \"//build_defs:test\",\n" +
				")",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := newExporter(nil, "", false).(*defaultExporter)

			pkg := &core.Package{Name: "test"}
			e.requiredSubincludes[pkg] = map[core.BuildLabel]bool{}
			for _, label := range tc.requiredLabels {
				e.requiredSubincludes[pkg][label] = true
			}

			assert.Equal(t, tc.out, e.minimalSubincludeStatement(pkg, tc.availableLabels))
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

	// stmtIndices maps target names to their statement index in filter.build
	stmtIndices := map[string]int{
		"a": 2,
		"b": 3,
	}

	targetLabels := map[string]core.BuildLabel{}
	for name, index := range stmtIndices {
		label := core.NewBuildLabel("test", name)
		targetLabels[name] = label
		target := &core.BuildTarget{Label: label}
		pkg.Metadata.RegisterStatementTarget(target, func() core.BuildStatement {
			return asp.NewBuildStatement(statements[index])
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			caseTargets := map[core.BuildLabel]bool{}
			for _, name := range tc.required {
				caseTargets[targetLabels[name]] = true
			}

			e := newExporter(nil, "", false).(*defaultExporter)
			e.exportedTargets[pkg] = caseTargets

			got, err := e.filterPackageFile(pkg)
			assert.NoError(t, err)

			expected, err := os.ReadFile(tc.expected)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), string(got))
		})
	}
}
