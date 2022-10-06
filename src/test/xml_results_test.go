package test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestParseJUnitXMLResultsOneSuccessfulTest(t *testing.T) {
	sample := bytes.NewBufferString("<testcase name=\"case\" time=\"0.5\"></testcase>").Bytes()
	testSuites, err := parseJUnitXMLTestResults(sample)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(testSuites.TestSuites))
	assert.Equal(t, 500*time.Millisecond, testSuites.TestSuites[0].Duration)

	testSuite := testSuites.TestSuites[0]

	assert.Equal(t, 1, len(testSuite.TestCases))
	assert.Equal(t, 500*time.Millisecond, testSuite.Duration)

	testCase := testSuite.TestCases[0]

	assert.NotNil(t, testCase.Success())
	assert.Equal(t, 500*time.Millisecond, *testCase.Duration())
}

func TestUpload(t *testing.T) {
	results := map[string][]byte{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		results[r.URL.Path] = b
	}))
	target := xmlTestScenario()

	err := uploadResults(target, s.URL+"/results", false, false)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(results["/results"]))

	err = uploadResults(target, s.URL+"/results_success_output", false, true)
	assert.NoError(t, err)
	assert.Equal(t, expectedWithSuccessOutput, string(results["/results_success_output"]))
}

func TestUploadGzipped(t *testing.T) {
	results := map[string][]byte{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := gzip.NewReader(r.Body)
		assert.NoError(t, err)
		b, err := io.ReadAll(body)
		assert.NoError(t, err)
		results[r.URL.Path] = b
	}))

	target := xmlTestScenario()

	err := uploadResults(target, s.URL+"/results", true, false)
	assert.NoError(t, err)
	assert.Equal(t, []byte(expected), results["/results"])
}

func xmlTestScenario() *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/core:lock_test", ""))
	duration := 500 * time.Millisecond
	target.Test = new(core.TestFields)
	target.Test.Results = &core.TestSuite{
		Package:  "src.core",
		Name:     "lock_test",
		Duration: 1 * time.Second,
		TestCases: core.TestCases{
			{
				ClassName: "src.core.lock_test",
				Name:      "TestAcquireRepoLock",
				Executions: []core.TestExecution{
					{
						Duration: &duration,
						Failure:  &core.TestResultFailure{},
						Stdout:   "failure out",
					},
					{
						Duration: &duration,
						Stdout:   "success out",
					},
				},
			},
			{
				ClassName: "src.core.lock_test",
				Name:      "TestReadLastOperationFailure",
				Executions: []core.TestExecution{
					{
						Duration: &duration,
						Failure:  &core.TestResultFailure{},
						Stdout:   "failure out",
					},
				},
			},
			{
				ClassName: "src.core.lock_test",
				Name:      "TestReadLastOperation",
				Executions: []core.TestExecution{
					{
						Duration: &duration,
						Stdout:   "out",
					},
				},
			},
		},
	}
	return target
}

const expected = `<testsuites name="//src/core:lock_test" time="1">
    <testsuite name="lock_test" tests="3" failures="1" package="src.core" time="1">
        <properties></properties>
        <testcase name="TestAcquireRepoLock" classname="src.core.lock_test" time="0.5">
            <flakyFailure type="">
                <system-out>failure out</system-out>
            </flakyFailure>
        </testcase>
        <testcase name="TestReadLastOperationFailure" classname="src.core.lock_test" time="0.5">
            <failure type=""></failure>
            <system-out>failure out</system-out>
        </testcase>
        <testcase name="TestReadLastOperation" classname="src.core.lock_test" time="0.5"></testcase>
    </testsuite>
</testsuites>`

const expectedWithSuccessOutput = `<testsuites name="//src/core:lock_test" time="1">
    <testsuite name="lock_test" tests="3" failures="1" package="src.core" time="1">
        <properties></properties>
        <testcase name="TestAcquireRepoLock" classname="src.core.lock_test" time="0.5">
            <flakyFailure type="">
                <system-out>failure out</system-out>
            </flakyFailure>
            <system-out>success out</system-out>
        </testcase>
        <testcase name="TestReadLastOperationFailure" classname="src.core.lock_test" time="0.5">
            <failure type=""></failure>
            <system-out>failure out</system-out>
        </testcase>
        <testcase name="TestReadLastOperation" classname="src.core.lock_test" time="0.5">
            <system-out>out</system-out>
        </testcase>
    </testsuite>
</testsuites>`
