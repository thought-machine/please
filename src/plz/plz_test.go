package plz

import (
	"fmt"
	"github.com/thought-machine/please/src/cli"
	"testing"

	"github.com/thought-machine/please/src/core"

	"github.com/stretchr/testify/assert"
)

func TestStripHost(t *testing.T) {
	hostArch := cli.HostArch()
	testData := []struct{
		Name string
		Plugin string
		Label string
		Expected string
	} {
		{
			Name: "Leaves no subrepo",
			Label: "//foo:bar",
			Expected: "//foo:bar",
		},
		{
			Name: "Strips host arch",
			Label: fmt.Sprintf("@%v//foo:bar", hostArch.String()),
			Expected: "//foo:bar",
		},
		{
			Name: "Leaves target arch",
			Label: fmt.Sprintf("@%v//foo:bar", "foo_bar64"),
			Expected: "@foo_bar64//foo:bar",
		},
		{
			Name: "Strips plugin repo",
			Label: "@go//foo:bar",
			Plugin: "go",
			Expected: "//foo:bar",
		},
		{
			Name: "Leaves different subrepo",
			Label: "@python//foo:bar",
			Plugin: "go",
			Expected: "@python//foo:bar",
		},
		{
			Name: "Strips host plugin repo",
			Label: fmt.Sprintf("@%v//foo:bar", core.SubrepoArchName("go", cli.HostArch())),
			Plugin: "go",
			Expected: "//foo:bar",
		},
		{
			Name: "Strips host but leaves different subrepo",
			Label: fmt.Sprintf("@%v//foo:bar", core.SubrepoArchName("python", cli.HostArch())),
			Plugin: "go",
			Expected: "@python//foo:bar",
		},

	}

	for _, test := range testData {
		t.Run(test.Name, func(t *testing.T) {
			config := core.DefaultConfiguration()
			config.PluginDefinition.Name = test.Plugin
			actual := stripHostRepoName(config, core.ParseBuildLabel(test.Label, ""))
			assert.Equal(t, core.ParseBuildLabel(test.Expected,  ""), actual)
		})
	}
}