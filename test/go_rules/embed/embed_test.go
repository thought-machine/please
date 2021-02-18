package embed

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLibEmbed(t *testing.T) {
	assert.Equal(t, "hello", strings.TrimSpace(hello))
}
