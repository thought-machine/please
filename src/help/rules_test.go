package help

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestRuleArgs(t *testing.T) {
	funcMap := getFunctionsFromState(core.NewDefaultBuildState())
	env := getRuleArgs(funcMap)
	assert.True(t, len(env.Functions) > 20) // Don't care exactly how many there are, but it should have a fair few.
	rule := env.Functions["cc_library"]
	assert.True(t, len(rule.Args) > 5)
	assert.NotEqual(t, "", rule.Comment)
	assert.NotEqual(t, "", rule.Docstring)
	// Some of this is getting a bit more specific than I'd like, but we have to test it on *something*,
	// and it'd not be hard to update if the rule does change.
	arg := rule.Args[0]
	assert.Equal(t, "name", arg.Name)
	assert.True(t, arg.Required)
	assert.False(t, arg.Deprecated)
	assert.Equal(t, []string{"str"}, arg.Types)
	assert.Equal(t, "Name of the rule", arg.Comment)
	arg = rule.Args[3]
	assert.Equal(t, "hdrs", arg.Name)
	assert.False(t, arg.Required)
	assert.False(t, arg.Deprecated)
	assert.Equal(t, []string{"list"}, arg.Types)
	assert.Equal(t, "Header files. These will be made available to dependent rules, so the distinction between srcs and hdrs is important.", arg.Comment)
}

func TestMultilineComment(t *testing.T) {
	funcMap := getFunctionsFromState(core.NewDefaultBuildState())
	env := getRuleArgs(funcMap)
	rule := env.Functions["new_http_archive"]
	assert.True(t, strings.Count(rule.Comment, "\n") > 1)
}
