package cgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnswer(t *testing.T) {
	assert.Equal(t, 42, GetAnswer())
}

func TestQuestion(t *testing.T) {
	// There has always been something wrong with the world.
	assert.Equal(t, "What do you get if you multiply six by nine?", GetQuestion())
}
