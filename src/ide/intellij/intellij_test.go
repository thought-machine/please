package intellij

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestProjectLocation(t *testing.T) {
	assert.True(t, strings.HasSuffix(projectLocation(), "plz-out/intellij/.idea"))
}

func TestModuleFileLocation(t *testing.T) {
	label := core.BuildLabel{
		PackageName: "some/package", Name: "target", Subrepo: "",
	}

	assert.True(t, strings.HasSuffix(moduleFileLocation(label), "plz-out/intellij/some/package/some_package_target.iml"))
}
