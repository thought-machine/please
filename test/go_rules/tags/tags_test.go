package tags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuestion(t *testing.T) {
	assert.Equal(t, "what do you get if you multiply six by seven", WhatIsTheQuestion())
}
