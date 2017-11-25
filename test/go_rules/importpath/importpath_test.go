package importpath_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/test/go_rules/importpath"
)

func TestTheAnswer(t *testing.T) {
	// This is a pretty silly test, but it's really about whether this compiles or not.
	assert.Equal(t, 42, importpath.Answer)
}
