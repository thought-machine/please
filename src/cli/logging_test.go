package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLineWrap(t *testing.T) {
	backend := NewLogBackend(2)
	backend.Cols = 80
	backend.maxLines = 2

	s := backend.lineWrap(strings.Repeat("a", 40))
	assert.Equal(t, strings.Repeat("a", 40), strings.Join(s, "\n"))

	s = backend.lineWrap(strings.Repeat("a", 100))
	assert.Equal(t, strings.Repeat("a", 20)+"\n"+strings.Repeat("a", 80), strings.Join(s, "\n"))

	s = backend.lineWrap(strings.Repeat("a", 80))
	assert.Equal(t, strings.Repeat("a", 80), strings.Join(s, "\n"))
}
