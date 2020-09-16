package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)



func TestParse(t *testing.T) {
	cmd := parse("echo $(basename $(location //foo/bar)) > $OUT", "/")

	assert.Len(t, cmd.tokens, 3)
	assert.Equal(t, bash("echo $(basename "), cmd.tokens[0])
	assert.Equal(t, location{PackageName: "foo/bar", Name: "bar"}, cmd.tokens[1])
	assert.Equal(t, bash(") > $OUT"), cmd.tokens[2])
}