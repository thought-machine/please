package asp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolvePluginTargetValues(t *testing.T) {
	values := resolvePluginValue([]string{"//path/to:target"}, "path/to/subrepo")
	assert.Equal(t, []string{"///path/to/subrepo//path/to:target"}, values)

	values = resolvePluginValue([]string{"///path/to/another/plugin//path/to:target"}, "path/to/subrepo")
	assert.Equal(t, []string{"///path/to/another/plugin//path/to:target"}, values)

	values = resolvePluginValue([]string{"///path/to/another/plugin//path/to:target"}, "")
	assert.Equal(t, []string{"///path/to/another/plugin//path/to:target"}, values)

	values = resolvePluginValue([]string{"//path/to:target"}, "")
	assert.Equal(t, []string{"/////path/to:target"}, values)
}
