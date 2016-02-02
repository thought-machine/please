package core

import "testing"

var a = []LineCoverage{NotExecutable, Uncovered, Uncovered, Covered, NotExecutable, Unreachable}
var b = []LineCoverage{Uncovered, Covered, Uncovered, Uncovered, Unreachable, Covered}
var c = []LineCoverage{Covered, NotExecutable, Covered, Uncovered, Covered, NotExecutable}
var empty = []LineCoverage{}

func assertCoverage(t *testing.T, expected, coverage []LineCoverage) {
	if len(coverage) != len(expected) {
		t.Errorf("Incorrect length of produced coverage; should be %d, was %d", len(expected), len(coverage))
	} else {
		for i, cvr := range coverage {
			if expected[i] != cvr {
				t.Errorf("Incorrect coverage entry at index %d: should be %d, was %d", i, expected[i], cvr)
			}
		}
	}
}

func TestMergeCoverageLines1(t *testing.T) {
	coverage := MergeCoverageLines(a, b)
	expected := []LineCoverage{Uncovered, Covered, Uncovered, Covered, Unreachable, Covered}
	assertCoverage(t, expected, coverage)
}

func TestMergeCoverageLines2(t *testing.T) {
	coverage := MergeCoverageLines(a, c)
	expected := []LineCoverage{Covered, Uncovered, Covered, Covered, Covered, Unreachable}
	assertCoverage(t, expected, coverage)
}

func TestMergeCoverageLines3(t *testing.T) {
	coverage := MergeCoverageLines(b, c)
	expected := []LineCoverage{Covered, Covered, Covered, Uncovered, Covered, Covered}
	assertCoverage(t, expected, coverage)
}

func TestMergeCoverageLines4(t *testing.T) {
	coverage := MergeCoverageLines(MergeCoverageLines(a, b), c)
	expected := []LineCoverage{Covered, Covered, Covered, Covered, Covered, Covered}
	assertCoverage(t, expected, coverage)
}

func TestMergeCoverageLines5(t *testing.T) {
	coverage := MergeCoverageLines(MergeCoverageLines(c, a), b)
	expected := []LineCoverage{Covered, Covered, Covered, Covered, Covered, Covered}
	assertCoverage(t, expected, coverage)
}

func TestMergeCoverageLines6(t *testing.T) {
	coverage := MergeCoverageLines(empty, b)
	assertCoverage(t, b, coverage)
}

func TestMergeCoverageLines7(t *testing.T) {
	coverage := MergeCoverageLines(a, empty)
	assertCoverage(t, a, coverage)
}

func TestMergeCoverageLines8(t *testing.T) {
	coverage := MergeCoverageLines(empty, empty)
	assertCoverage(t, empty, coverage)
}
