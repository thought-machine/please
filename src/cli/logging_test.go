package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/op/go-logging.v1"
)

func TestLineWrap(t *testing.T) {
	newLogBackend(logging.NewLogBackend(os.Stderr, "", 0))
	backend := CurrentBackend
	backend.cols = 80
	backend.maxLines = 2
	backend.interactiveRows = 2
	backend.recalcWindowSize()

	s := backend.lineWrap(strings.Repeat("a", 40))
	assert.Equal(t, strings.Repeat("a", 40), strings.Join(s, "\n"))

	s = backend.lineWrap(strings.Repeat("a", 100))
	assert.Equal(t, strings.Repeat("a", 20)+"\n"+strings.Repeat("a", 80), strings.Join(s, "\n"))

	s = backend.lineWrap(strings.Repeat("a", 80))
	assert.Equal(t, strings.Repeat("a", 80), strings.Join(s, "\n"))
}

func TestParseVerbosity(t *testing.T) {
	var v Verbosity
	assert.NoError(t, v.UnmarshalFlag("error"))
	assert.EqualValues(t, logging.ERROR, v)
	assert.NoError(t, v.UnmarshalFlag("1"))
	assert.EqualValues(t, logging.WARNING, v)
	assert.NoError(t, v.UnmarshalFlag("v"))
	assert.EqualValues(t, logging.NOTICE, v)
	assert.Error(t, v.UnmarshalFlag("blah"))
}
