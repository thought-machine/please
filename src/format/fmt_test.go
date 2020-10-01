package format

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

func TestFormat(t *testing.T) {
	const before = "src/fmt/test_data/before.build"
	const after = "src/fmt/test_data/after.build"

	changed, err := Format(core.DefaultConfiguration(), []string{"src/fmt/test_data/before.build"}, false)
	assert.NoError(t, err)
	assert.True(t, changed)

	// N.B. this rewrites the file; be careful if you're adding more tests here.
	changed, err = Format(core.DefaultConfiguration(), []string{before}, true)
	assert.NoError(t, err)
	assert.True(t, changed)

	beforeContents, err := ioutil.ReadFile(before)
	require.NoError(t, err)
	afterContents, err := ioutil.ReadFile(after)
	require.NoError(t, err)
	assert.Equal(t, beforeContents, afterContents)
}
