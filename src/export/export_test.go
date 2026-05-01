package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thought-machine/please/src/core"
)

func TestMinimalSubincludeStatement(t *testing.T) {
	var subincludesTests = []struct {
		name            string
		availableLabels []core.BuildLabel
		requiredLabels  []core.BuildLabel
		out             string
	}{
		{
			"Successful no pruning subinclude",
			core.ParseBuildLabels([]string{"//build_defs:test"}),
			core.ParseBuildLabels([]string{"//build_defs:test"}),
			`subinclude("//build_defs:test")`,
		},
		{
			"No subincludes",
			nil,
			nil,
			"",
		},
		{
			"Single subinclude (not required)",
			core.ParseBuildLabels([]string{"//build_defs:other"}),
			nil,
			"",
		},
		{
			"Multiple subincludes (sorted and filtered)",
			core.ParseBuildLabels([]string{"//build_defs:test", "//build_defs:abc", "//build_defs:other"}),
			core.ParseBuildLabels([]string{"//build_defs:test", "//build_defs:abc"}),
			"subinclude(\n" +
				"    \"//build_defs:abc\",\n" +
				"    \"//build_defs:test\",\n" +
				")",
		},
	}

	for _, tt := range subincludesTests {
		t.Run(tt.name, func(t *testing.T) {
			e := &DefaultExporter{
				requiredSubincludes: map[*core.Package]map[core.BuildLabel]bool{},
			}

			pkg := &core.Package{Name: "test"}
			e.requiredSubincludes[pkg] = map[core.BuildLabel]bool{}
			for _, labels := range tt.requiredLabels {
				e.requiredSubincludes[pkg][labels] = true
			}

			assert.Equal(t, tt.out, e.minimalSubincludeStatement(pkg, tt.availableLabels))
		})
	}
}
