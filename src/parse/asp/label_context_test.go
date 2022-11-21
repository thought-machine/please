package asp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

func newScope(pkgName, subrepo, plugin string) *scope {
	s := &scope{
		pkg:   core.NewPackageSubrepo(pkgName, subrepo),
		state: core.NewBuildState(core.DefaultConfiguration()),
	}
	if plugin != "" {
		s.state.Config.PluginDefinition.Name = plugin
	}
	return s
}

func TestParseLabelContext(t *testing.T) {
	arch := cli.HostArch()

	testCases := []struct {
		testName           string
		label              string
		scope              *scope
		subrepo, pkg, name string
	}{
		{
			testName: "Test parse absolute label with subrepo using @",
			label:    "@other_subrepo//test:target",
			scope:    newScope("subrepo_package", "subrepo", ""),
			subrepo:  "other_subrepo",
			pkg:      "test",
			name:     "target",
		},
		{
			testName: "Test parse absolute label with subrepo using ///",
			label:    "///other_subrepo//test:target",
			scope:    newScope("subrepo_package", "subrepo", ""),
			subrepo:  "other_subrepo",
			pkg:      "test",
			name:     "target",
		},
		{
			testName: "Test host reference using @",
			label:    "@//test:target",
			scope:    newScope("subrepo_package", "subrepo", ""),
			subrepo:  "",
			pkg:      "test",
			name:     "target",
		},
		{
			testName: "Test host reference using ///",
			label:    "/////test:target",
			scope:    newScope("subrepo_package", "subrepo", ""),
			subrepo:  "",
			pkg:      "test",
			name:     "target",
		},
		{
			testName: "Test label relative to subrepo",
			label:    "//test:target",
			scope:    newScope("subrepo_package", "subrepo", ""),
			subrepo:  "subrepo",
			pkg:      "test",
			name:     "target",
		},
		{
			testName: "Test label relative to package in subrepo",
			label:    ":target",
			scope:    newScope("subrepo_package", "subrepo", ""),
			subrepo:  "subrepo",
			pkg:      "subrepo_package",
			name:     "target",
		},
		{
			testName: "Test host arch is stripped",
			label:    fmt.Sprintf("///%s//test:target", (&arch).String()),
			scope:    newScope("pkg", "", ""),
			subrepo:  "",
			pkg:      "test",
			name:     "target",
		},
		{
			testName: "Test host arch is stripped from subrepo",
			label:    fmt.Sprintf("///subrepo_%s//test:target", (&arch).String()),
			scope:    newScope("pkg", "", ""),
			subrepo:  "subrepo",
			pkg:      "test",
			name:     "target",
		},
		{
			testName: "Test host plugin is stripped",
			label:    "///foo//test:target",
			scope:    newScope("subrepo_package", "", "foo"),
			subrepo:  "",
			pkg:      "test",
			name:     "target",
		},
	}

	for _, test := range testCases {
		t.Run(test.testName, func(t *testing.T) {
			label := test.scope.parseLabelInPackage(test.label, test.scope.contextPackage())
			assert.Equal(t, test.subrepo, label.Subrepo)
			assert.Equal(t, test.pkg, label.PackageName)
			assert.Equal(t, test.name, label.Name)
		})
	}
}
