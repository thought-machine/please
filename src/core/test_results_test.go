package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var a = []LineCoverage{NotExecutable, Uncovered, Uncovered, Covered, NotExecutable, Unreachable}
var b = []LineCoverage{Uncovered, Covered, Uncovered, Uncovered, Unreachable, Covered}
var c = []LineCoverage{Covered, NotExecutable, Covered, Uncovered, Covered, NotExecutable}
var empty = []LineCoverage{}

func TestMergeCoverageLines1(t *testing.T) {
	coverage := MergeCoverageLines(a, b)
	expected := []LineCoverage{Uncovered, Covered, Uncovered, Covered, Unreachable, Covered}
	assert.Equal(t, expected, coverage)
}

func TestMergeCoverageLines2(t *testing.T) {
	coverage := MergeCoverageLines(a, c)
	expected := []LineCoverage{Covered, Uncovered, Covered, Covered, Covered, Unreachable}
	assert.Equal(t, expected, coverage)
}

func TestMergeCoverageLines3(t *testing.T) {
	coverage := MergeCoverageLines(b, c)
	expected := []LineCoverage{Covered, Covered, Covered, Uncovered, Covered, Covered}
	assert.Equal(t, expected, coverage)
}

func TestMergeCoverageLines4(t *testing.T) {
	coverage := MergeCoverageLines(MergeCoverageLines(a, b), c)
	expected := []LineCoverage{Covered, Covered, Covered, Covered, Covered, Covered}
	assert.Equal(t, expected, coverage)
}

func TestMergeCoverageLines5(t *testing.T) {
	coverage := MergeCoverageLines(MergeCoverageLines(c, a), b)
	expected := []LineCoverage{Covered, Covered, Covered, Covered, Covered, Covered}
	assert.Equal(t, expected, coverage)
}

func TestMergeCoverageLines6(t *testing.T) {
	coverage := MergeCoverageLines(empty, b)
	assert.Equal(t, b, coverage)
}

func TestMergeCoverageLines7(t *testing.T) {
	coverage := MergeCoverageLines(a, empty)
	assert.Equal(t, a, coverage)
}

func TestMergeCoverageLines8(t *testing.T) {
	coverage := MergeCoverageLines(empty, empty)
	assert.Equal(t, empty, coverage)
}
