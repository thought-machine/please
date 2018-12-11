package test_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/thought-machine/please/test/go_rules/test"
)

func TestAnswer(t *testing.T) {
	assert.Equal(t, 42, GetAnswer())
	assert.Equal(t, "var", GetVar())
	assert.Equal(t, "var1 var2", GetVar2())

	const featureAdded = 1544432014
	lastCommitTime, err := strconv.ParseInt(GetExecGitShow(), 10, 64)
	if !assert.NoError(t, err) {
		assert.Fail(t, "unable to parse time")
	}
	assert.True(t, lastCommitTime > featureAdded, "git_show(): time went backwards")

	assert.Contains(t, GetExecGitState(), "shiny", "git_state(): failed")

	assert.Len(t, GetExecGitCommitFull(), 40, "git_commit() full length wrong")
	assert.Len(t, GetExecGitCommitShort(), 8, "git_commit() short length wrong")
}
