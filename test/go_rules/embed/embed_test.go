package embed

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLibEmbed(t *testing.T) {
	assert.Equal(t, "hello", hello)
}
