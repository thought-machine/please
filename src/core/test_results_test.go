package core

import (
	"testing"
	"time"

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

func TestAdd(t *testing.T) {
	duration10 := time.Duration(10)
	duration20 := time.Duration(20)
	suite1 := TestSuite{
		Name: "Test",
		TestCases: []TestCase{
			{
				ClassName: "SomeClass",
				Name:      "someTest",
				Executions: []TestExecution{
					{
						Duration: &duration10,
					},
				},
			},
		},
	}
	suite2 := TestSuite{
		Name: "Test",
		TestCases: []TestCase{
			{
				ClassName: "SomeClass",
				Name:      "someTest",
				Executions: []TestExecution{
					{
						Duration: &duration20,
					},
				},
			},
			{
				ClassName: "SomeClass",
				Name:      "someTest2",
				Executions: []TestExecution{
					{
						Duration: &duration20,
					},
				},
			},
		},
	}
	suite1.Add(suite2.TestCases...)

	assert.Equal(t, 2, suite1.Tests())
	assert.Equal(t, 2, suite1.Passes())
	assert.Equal(t, 0, suite1.Failures())
	assert.Equal(t, 0, suite1.Skips())
	assert.Equal(t, 0, suite1.Errors())

	testCase := suite1.TestCases[0]

	assert.Equal(t, 2, len(testCase.Executions))
	assert.NotNil(t, testCase.Success())

	success := testCase.Success()

	assert.Equal(t, &duration10, success.Duration)

	testCase = suite1.TestCases[1]

	assert.Equal(t, 1, len(testCase.Executions))
	assert.NotNil(t, testCase.Success())

	success = testCase.Success()

	assert.Equal(t, &duration20, success.Duration)
}
