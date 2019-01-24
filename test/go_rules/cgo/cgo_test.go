package cgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnswer(t *testing.T) {
	assert.Equal(t, 42, GetAnswer())
}

func TestQuestion(t *testing.T) {
	assert.NoError(t, CheckAnswer("42"))
}
