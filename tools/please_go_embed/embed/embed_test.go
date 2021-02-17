package embed

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEmbed(t *testing.T) {
	cfg, err := Parse([]string{"tools/please_go_embed/embed/test_data/test.go"})
	assert.NoError(t, err)
	assert.Equal(t, map[string][]string{
		"hello.txt":   {"hello.txt"},
		"files/*.txt": {"files/test.txt"},
	}, cfg.Patterns)
	assert.Equal(t, map[string]string{
		"hello.txt":      "hello.txt",
		"files/test.txt": "files/test.txt",
	}, cfg.Files)
}
