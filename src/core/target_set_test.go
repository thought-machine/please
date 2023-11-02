package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTargetSetMatch(t *testing.T) {
	ts := NewTargetSet()
	ts.Add(ParseBuildLabel("//src/core:core", ""))
	ts.Add(ParseBuildLabel("//src/parse:all", ""))
	assert.True(t, ts.Match(ParseBuildLabel("//src/core:core", "")))
	assert.False(t, ts.Match(ParseBuildLabel("//src/core:core_test", "")))
	assert.True(t, ts.Match(ParseBuildLabel("//src/parse:parse", "")))
	assert.True(t, ts.Match(ParseBuildLabel("//src/parse:parse_test", "")))
	assert.False(t, ts.Match(ParseBuildLabel("//src/build", "")))
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
