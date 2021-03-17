package test

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestParseJUnitXMLResults_oneSuccessfulTest(t *testing.T) {
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
		b, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		results[r.URL.Path] = b
	}))
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/core:lock_test", ""))
	duration := 500 * time.Millisecond
	target.Results = core.TestSuite{
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
					},
				},
			},
			{
				ClassName: "src.core.lock_test",
				Name:      "TestReadLastOperation",
				Executions: []core.TestExecution{
					{
						Duration: &duration,
					},
				},
			},
		},
	}
	target.IsTest = true

	err := uploadResults(target, s.URL+"/results", false)
	assert.NoError(t, err)
	assert.Equal(t, []byte(expected), results["/results"])
}

func TestUploadGzipped(t *testing.T) {
	results := map[string][]byte{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := gzip.NewReader(r.Body)
		assert.NoError(t, err)
		b, err := ioutil.ReadAll(body)
		assert.NoError(t, err)
		results[r.URL.Path] = b
	}))
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/core:lock_test", ""))
	duration := 500 * time.Millisecond
	target.Results = core.TestSuite{
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
					},
				},
			},
			{
				ClassName: "src.core.lock_test",
				Name:      "TestReadLastOperation",
				Executions: []core.TestExecution{
					{
						Duration: &duration,
					},
				},
			},
		},
	}
	target.IsTest = true

	err := uploadResults(target, s.URL+"/results", true)
	assert.NoError(t, err)
	assert.Equal(t, []byte(expected), results["/results"])
}

const expected = `<testsuites name="//src/core:lock_test" time="1">
    <testsuite name="lock_test" tests="2" package="src.core" time="1">
        <properties></properties>
        <testcase name="TestAcquireRepoLock" classname="src.core.lock_test" time="0.5"></testcase>
        <testcase name="TestReadLastOperation" classname="src.core.lock_test" time="0.5"></testcase>
    </testsuite>
</testsuites>`
