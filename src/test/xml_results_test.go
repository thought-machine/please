package test

import (
	"testing"
	"bytes"

	"github.com/stretchr/testify/assert"
	"time"
)

func TestParseJUnitXMLResults_oneSuccessfulTest(t *testing.T) {
	sample := bytes.NewBufferString("<testcase name=\"case\" time=\"0.5\"></testcase>").Bytes()
	testSuites, err := parseJUnitXMLTestResults(sample)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t,1, len(testSuites.TestSuites))
	assert.Equal(t, time.Duration(500 * time.Millisecond), testSuites.TestSuites[0].Duration)

	testSuite := testSuites.TestSuites[0]

	assert.Equal(t, 1, len(testSuite.TestCases))
	assert.Equal(t, time.Duration(500 * time.Millisecond), testSuite.Duration)

	testCase := testSuite.TestCases[0]

	assert.NotNil(t, testCase.Success())
	assert.Equal(t, time.Duration(500 * time.Millisecond), *testCase.Duration())
}
