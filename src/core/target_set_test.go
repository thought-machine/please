package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTargetSetMatch(t *testing.T) {
	ts := NewTargetSet()
	ts.Add(ParseBuildLabel("//src/core:core", ""))
	ts.Add(ParseBuildLabel("//src/parse:all", ""))
	matched, exact := ts.Match(ParseBuildLabel("//src/core:core", ""))
	assert.True(t, matched)
	assert.True(t, exact)
	matched, exact = ts.Match(ParseBuildLabel("//src/core:core_test", ""))
	assert.False(t, matched)
	assert.False(t, exact)
	matched, exact = ts.Match(ParseBuildLabel("//src/parse:parse", ""))
	assert.True(t, matched)
	assert.False(t, exact)
	matched, exact = ts.Match(ParseBuildLabel("//src/parse:parse_test", ""))
	assert.True(t, matched)
	assert.False(t, exact)
	matched, exact = ts.Match(ParseBuildLabel("//src/build", ""))
	assert.False(t, matched)
	assert.False(t, exact)
}

func TestTargetSetMatchExact(t *testing.T) {
	ts := NewTargetSet()
	ts.Add(ParseBuildLabel("//src/core:core", ""))
	ts.Add(ParseBuildLabel("//src/parse:all", ""))
	assert.True(t, ts.MatchExact(ParseBuildLabel("//src/core:core", "")))
	assert.False(t, ts.MatchExact(ParseBuildLabel("//src/core:core_test", "")))
	assert.False(t, ts.MatchExact(ParseBuildLabel("//src/parse:parse", "")))
	assert.False(t, ts.MatchExact(ParseBuildLabel("//src/parse:parse_test", "")))
	assert.False(t, ts.MatchExact(ParseBuildLabel("//src/build", "")))
}

func TestAllTargets(t *testing.T) {
	ts := NewTargetSet()
	labels := []BuildLabel{
		ParseBuildLabel("//src/core:core", ""),
		ParseBuildLabel("//src/core:core_test", ""),
		ParseBuildLabel("//src/parse:all", ""),
		ParseBuildLabel("//src/parse:parse_test", ""),
	}
	for _, label := range labels {
		ts.Add(label)
	}
	assert.Equal(t, labels, ts.AllTargets())
}
