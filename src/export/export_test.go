package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thought-machine/please/src/core"
)

func TestMakeSubincludesStatement(t *testing.T) {
	e := &export{
		requiredSubincludes: map[*core.Package]map[core.BuildLabel]bool{},
	}

	pkg := &core.Package{Name: "test"}

	// Test case 1: No subincludes
	assert.Equal(t, "", e.makeSubincludesStatement(pkg))

	// Test case 2: Single subinclude
	label1 := core.ParseBuildLabel("//build_defs:test", "")
	e.requiredSubincludes[pkg] = map[core.BuildLabel]bool{
		label1: true,
	}
	assert.Equal(t, `subinclude("//build_defs:test")`, e.makeSubincludesStatement(pkg))

	// Test case 3: Multiple subincludes (sorted)
	label2 := core.ParseBuildLabel("//build_defs:abc", "")
	e.requiredSubincludes[pkg][label2] = true
	expected := "subinclude(\n    \"//build_defs:abc\",\n    \"//build_defs:test\",\n)"
	assert.Equal(t, expected, e.makeSubincludesStatement(pkg))
}
