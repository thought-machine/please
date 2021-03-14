package glob

import (
	"fmt"
	"github.com/stretchr/testify/assert"

	"strings"
	"testing"
)

func evalPattern(m *matcher, path []string) bool {
	if len(path) == 0 {
		return false
	}
	matched, next := m.match(path[0])
	if next == nil {
		return matched
	}
	return matched || evalPattern(next, path[1:])
}

func TestIsGlob(t *testing.T) {
	assert.True(t, IsGlob("a*b"))
	assert.True(t, IsGlob("ab/*.txt"))
	assert.True(t, IsGlob("ab/c.tx?"))
	assert.True(t, IsGlob("ab/[a-z].txt"))
	assert.False(t, IsGlob("abc.txt"))
	assert.False(t, IsGlob("ab/c.txt"))
}

func TestCompilePattern(t *testing.T) {
	testCases := []struct {
		name          string
		pattern       string
		shouldMatch   string
		shouldntMatch string
	}{
		{
			name:          "exact match",
			pattern:       "test.txt",
			shouldMatch:   "test.txt",
			shouldntMatch: "test.py",
		},
		{
			name:          "range match",
			pattern:       "[a-z]est.txt",
			shouldMatch:   "test.txt",
			shouldntMatch: "Test.txt",
		},
		{
			name:          "wild card match",
			pattern:       "?est.txt",
			shouldMatch:   "test.txt",
			shouldntMatch: "teest.txt",
		},
		{
			name:          "astrix matches one char",
			pattern:       "*est.txt",
			shouldMatch:   "test.txt",
			shouldntMatch: "test.py",
		},
		{
			name:          "astrix matches many char",
			pattern:       "*.txt",
			shouldMatch:   "test.txt",
			shouldntMatch: "test.py",
		},
		{
			name:          "double astrix matches same dir",
			pattern:       "**.txt",
			shouldMatch:   "test.txt",
			shouldntMatch: "test.py",
		},
		{
			name:          "double astrix matches sub dir",
			pattern:       "**.txt",
			shouldMatch:   "a/test.txt",
			shouldntMatch: "a/test.py",
		},
		{
			name:          "**/*.txt matches sub dir",
			pattern:       "**/*.txt",
			shouldMatch:   "a/test.txt",
			shouldntMatch: "test.test",
		},
		{
			name:          "+ matched literally",
			pattern:       "a/a+.txt",
			shouldMatch:   "a/a+.txt",
			shouldntMatch: "a/aa.txt",
		},
		{
			name:          "*thing* matches anything with that in file name",
			pattern:       "a/*.thing*",
			shouldMatch:   "a/some.thing.txt",
			shouldntMatch: "a/aa.txt",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			m := &matcher{
				includes: toPatterns([]string{testCase.pattern}, false),
			}

			matched := evalPattern(m, strings.Split(testCase.shouldMatch, "/"))
			assert.True(t, matched, fmt.Sprintf("%s should match %s", testCase.pattern, testCase.shouldMatch))

			matched = evalPattern(m, strings.Split(testCase.shouldntMatch, "/"))
			assert.False(t, matched, fmt.Sprintf("%s shouldn't match %s", testCase.pattern, testCase.shouldntMatch))
		})
	}
}