package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcNumRuns(t *testing.T) {
	// Helper for assert
	nr := func(a, b int) []interface{} { return []interface{}{a, b} }

	// Check that multiplication works.
	assert.Equal(t, nr(1, 1), nr(calcNumRuns(1, 1)))
	assert.Equal(t, nr(3, 1), nr(calcNumRuns(1, 3)))
	assert.Equal(t, nr(9, 3), nr(calcNumRuns(3, 3)))
	assert.Equal(t, nr(18, 6), nr(calcNumRuns(6, 3)))
	assert.Equal(t, nr(28, 7), nr(calcNumRuns(7, 4)))
}
