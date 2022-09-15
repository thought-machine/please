package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringifyAnnotatedOutputLabel(t *testing.T) {
	l := AnnotatedOutputLabel{BuildLabel: BuildLabel{PackageName: "src/core", Name: "build_input_test"}}
	assert.Equal(t, "//src/core:build_input_test", l.String())
	l.Annotation = "thing"
	assert.Equal(t, "//src/core:build_input_test|thing", l.String())
}
