package output

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColouriseError(t *testing.T) {
	err := fmt.Errorf("/opt/tm/toolchains/1.8.2/usr/include/fst/label-reachable.h:176:39: error: non-const lvalue reference to type 'unordered_map<int, int>' cannot bind to a value of unrelated type 'unordered_map<unsigned char, unsigned char>'")
	expected := fmt.Errorf("${BOLD_WHITE}/opt/tm/toolchains/1.8.2/usr/include/fst/label-reachable.h, line 176, column 39:${RESET} ${BOLD_RED}error: ${RESET}${BOLD_WHITE}non-const lvalue reference to type 'unordered_map<int, int>' cannot bind to a value of unrelated type 'unordered_map<unsigned char, unsigned char>'${RESET}")
	assert.EqualValues(t, expected, colouriseError(err))
}

func TestShouldInclude(t *testing.T) {
	testCases := []struct {
		testName        string
		includeFiles    []string
		file            string
		expectedOutcome bool
	}{
		{
			testName:        "with empty file list",
			includeFiles:    []string{},
			file:            "opt/tm/file.go",
			expectedOutcome: true,
		},
		{
			testName:        "with matching file",
			includeFiles:    []string{"opt/tm/file.go"},
			file:            "opt/tm/file.go",
			expectedOutcome: true,
		},
		{
			testName:        "with wildcard match",
			includeFiles:    []string{"opt/tm/*"},
			file:            "opt/tm/file.go",
			expectedOutcome: true,
		},
		{
			testName:        "with no matching files",
			includeFiles:    []string{"opt/tm/foo.go", "opt/tm/wibble.go"},
			file:            "opt/tm/file.go",
			expectedOutcome: false,
		},
		{
			testName:        "with nonsense input",
			includeFiles:    []string{"opt/tm/%^.&"},
			file:            "opt/tm/file.go",
			expectedOutcome: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			assert.Equal(t, testCase.expectedOutcome, shouldInclude(testCase.file, testCase.includeFiles))
		})
	}
}
