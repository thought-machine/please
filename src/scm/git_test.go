package scm

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseChangedLines(t *testing.T) {
	b, err := ioutil.ReadFile("src/scm/test_data/git.diff")
	assert.NoError(t, err)
	g := git{}
	m, err := g.parseChangedLines(b)
	assert.NoError(t, err)
	assert.Equal(t, map[string][]int{}, m)
}
