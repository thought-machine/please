package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thought-machine/please/src/core"
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

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			e := &defaultExporter{
				requiredSubincludes: map[*core.Package]map[core.BuildLabel]bool{},
			}

			pkg := &core.Package{Name: "test"}
			e.requiredSubincludes[pkg] = map[core.BuildLabel]bool{}
			for _, label := range test.requiredLabels {
				e.requiredSubincludes[pkg][label] = true
			}

			assert.Equal(t, test.out, e.minimalSubincludeStatement(pkg, test.availableLabels))
		})
	}
}
