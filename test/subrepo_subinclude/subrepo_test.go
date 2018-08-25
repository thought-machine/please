// Package test implements a test used to verify subincludes work on subrepos.
// The package name deliberately differs from the directory name in order to
// test some functionality of the subincluded rule.
package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmbeddedData(t *testing.T) {
	assert.EqualValues(t, []byte("kittens!\n"), MustAsset("test.txt"))
}
