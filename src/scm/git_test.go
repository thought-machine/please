package scm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseChangedLines(t *testing.T) {
	b, err := os.ReadFile("src/scm/test_data/git.diff")
	assert.NoError(t, err)
	g := git{}
	m, err := g.parseChangedLines(b)
	assert.NoError(t, err)
	assert.Equal(t, map[string][]int{
		"test/python_rules/behave/BUILD":                                      {8},
		"test/python_rules/behave/features/behave_test3.feature":              {1, 10},
		"test/python_rules/behave/features/steps/behave_test_steps.py":        {24, 25, 26, 27, 28},
		"test/python_rules/behave/features/test_suite_1/behave_test1.feature": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		"test/python_rules/behave/features/test_suite_2/behave_test2.feature": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		"tools/please_pex/behave.py":                                          {2, 3, 10, 11, 12, 13, 14, 15, 16, 17, 24, 25, 26, 27, 28, 29, 30, 31, 32},
	}, m)
}
