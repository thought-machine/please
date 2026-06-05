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
			e.requiredSubincludes[pkg.Label()] = tc.requiredLabels
			trimmer := trimmer{
				pkg:      pkg,
				exporter: e,
			}

			assert.Equal(t, tc.out, trimmer.minimalSubincludeStatement(tc.availableLabels))
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
			e := newExporter(nil, "", false).(*defaultExporter)
			for _, name := range tc.required {
				e.exportedTargets[targetLabels[name]] = true
			}
			e.visitedPackages[pkg.Label()] = true

			p := asp.NewParserOnly()
			got, err := e.trimPackage(p, pkg)
			assert.NoError(t, err)

			expected, err := os.ReadFile(tc.expected)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), string(got))
		})
	}
}

func TestStatementTrim(t *testing.T) {
	testCases := []struct {
		name       string
		content    string
		registered []string
		required   []string
		expected   string
	}{
		{
			name: "Keep target in if",
			content: `
if True:
  genrule(name = "a", cmd = "echo a > $OUT", outs = ["a"])
`,
			registered: []string{"a"},
			required:   []string{"a"},
			expected: `
if True:
  genrule(name = "a", cmd = "echo a > $OUT", outs = ["a"])
`,
		},
		{
			name: "Target not required - all statements trimmed",
			content: `
if True:
  genrule(name = "a", cmd = "echo a > $OUT", outs = ["a"])
`,
			registered: []string{"a"},
			required:   []string{},
			// Empty, all statements pruned. Blank space removal is not performed by trimBlock's implementation so expect the new lines.
			expected: `

`,
		},
		{
			name: "Required target in elif",
			content: `
if False:
    genrule(name = "a")
elif True:
    genrule(name = "b")
else:
    genrule(name = "c")
`,
			registered: []string{"b"},
			required:   []string{"b"},
			expected: `
if False:
    pass  #Trimmed during export
elif True:
    genrule(name = "b")
else:
    pass  #Trimmed during export
`},
		{
			name: "Required target in for",
			content: `
for i in range(0,2):
    genrule(name = "a")
`,
			registered: []string{"a"},
			required:   []string{"a"},
			expected: `
for i in range(0,2):
    genrule(name = "a")
`},
		{
			name: "Required if stmt in for",
			content: `
for i in [
    "a",
    "b",
]:
    if i == "a":
        genrule(name = "a")
    elif i == "b":
        genrule(name = "b")
`,
			registered: []string{"a", "b"},
			required:   []string{"a"},
			expected: `
for i in [
    "a",
    "b",
]:
    if i == "a":
        genrule(name = "a")
    elif i == "b":
        pass  #Trimmed during export
`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := asp.NewParserOnly()
			statements, err := p.ParseData([]byte(tc.content), "BUILD")
			assert.NoError(t, err)

			pkg := core.NewPackage("test", core.WithPackageMetadata())
			pkg.Filename = "BUILD"

			targetLabels := walkASTRegisterTargets(t, statements, pkg, tc.registered)
			e := newExporter(nil, "", false).(*defaultExporter)
			for _, name := range tc.required {
				e.exportedTargets[targetLabels[name]] = true
			}

			trimmer := &trimmer{
				origin:   []byte(tc.content),
				pkg:      pkg,
				exporter: e,
			}
			trimmer.trimBlock(statements, 0, asp.Position(len(tc.content)))

			assert.Equal(t, tc.expected, string(trimmer.bytes))
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

		// Not in targets we want to register, continue
		name := strings.Trim(arg.Value.Val.String, "\"")
		if toRegister != nil && !slices.Contains(toRegister, name) {
			return true
		}

		label := core.NewBuildLabel(pkg.Name, name)
		targetLabels[name] = label
		target := &core.BuildTarget{Label: label}
		pkg.Metadata.RegisterStatementTarget(target, func() core.BuildStatement {
			return asp.NewBuildStatement(stmt)
		})
		return true
	})
	return targetLabels
}
