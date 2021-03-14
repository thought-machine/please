package glob

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsGlob(t *testing.T) {
	assert.True(t, IsGlob("a*b"))
	assert.True(t, IsGlob("ab/*.txt"))
	assert.True(t, IsGlob("ab/c.tx?"))
	assert.True(t, IsGlob("ab/[a-z].txt"))
	assert.False(t, IsGlob("abc.txt"))
	assert.False(t, IsGlob("ab/c.txt"))
}
