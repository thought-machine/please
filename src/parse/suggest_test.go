package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestSuggestNoTargetFromSamePackage(t *testing.T) {
	pkg := makePackage("src/core", "wobble", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target2"), bl("//src/core:wibble"))
	assert.Equal(t, s, "", "No suggestion because they're not similar at all.")
}

func TestSuggestSingleTargetFromSamePackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target2"), bl("//src/core:wibble"))
	assert.Equal(t, s, "\nMaybe you meant :target1 ?")
}

func TestSuggestTwoTargetsFromSamePackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "target21", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target"), bl("//src/core:blibble"))
	assert.Equal(t, s, "\nMaybe you meant :target1 or :target21 ?")
}

func TestSuggestSeveralTargetsFromSamePackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "target21", "target_21", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target"), bl("//src/core:blibble"))
	assert.Equal(t, s, "\nMaybe you meant :target1 , :target21 or :target_21 ?")
}

func TestSuggestSingleTargetFromAnotherPackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target2"), bl("//src/parse:wibble"))
	assert.Equal(t, s, "\nMaybe you meant //src/core:target1 ?")
}

func TestSuggestTwoTargetsFromAnotherPackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "target21", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target"), bl("//src/parse:blibble"))
	assert.Equal(t, s, "\nMaybe you meant //src/core:target1 or //src/core:target21 ?")
}

func TestSuggestSeveralTargetsFromAnotherPackage(t *testing.T) {
	pkg := makePackage("src/core", "target1", "target21", "target_21", "wibble")
	s := suggestTargets(pkg, bl("//src/core:target"), bl("//src/parse:blibble"))
	assert.Equal(t, s, "\nMaybe you meant //src/core:target1 , //src/core:target21 or //src/core:target_21 ?")
}

func makePackage(name string, targets ...string) *core.Package {
	pkg := core.NewPackage(name)
	for _, target := range targets {
		pkg.AddTarget(core.NewBuildTarget(bl("//" + name + ":" + target)))
	}
	return pkg
}

func bl(label string) core.BuildLabel {
	return core.ParseBuildLabel(label, "")
}
