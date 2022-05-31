package embed

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLibEmbed(t *testing.T) {
	assert.Equal(t, "hello", strings.TrimSpace(hello))
}

func TestLibEmbedDir(t *testing.T) {
	b, err := testData.ReadFile("test_data/test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello world", strings.TrimSpace(string(b)))
}
