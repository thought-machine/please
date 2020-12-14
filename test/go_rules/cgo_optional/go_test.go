package cgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuestion(t *testing.T) {
	assert.NoError(t, CheckAnswer("42"))
}
