// Package asm_test implements an external test on compiling Go with assembly.
// It has to be external since right now we don't support Go tests with assembly sources.
package asm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/test/go_rules/asm"
)

func TestAssemblyAdd(t *testing.T) {
	assert.Equal(t, 42, asm.Add(40, 2))
}
