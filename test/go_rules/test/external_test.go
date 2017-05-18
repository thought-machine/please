package test_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	. "test/go_rules/test"
)

func TestAnswer(t *testing.T) {
	assert.Equal(t, 42, GetAnswer())
}
