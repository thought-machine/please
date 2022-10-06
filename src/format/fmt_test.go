package format

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

func TestFormat(t *testing.T) {
	const before = "src/format/test_data/before.build"
	const after = "src/format/test_data/after.build"

	changed, err := Format(core.DefaultConfiguration(), []string{before}, false, true)
	assert.NoError(t, err)
	assert.True(t, changed)

	// N.B. this rewrites the file; be careful if you're adding more tests here.
	changed, err = Format(core.DefaultConfiguration(), []string{before}, true, false)
	assert.NoError(t, err)
	assert.True(t, changed)

	beforeContents, err := os.ReadFile(before)
	require.NoError(t, err)
	afterContents, err := os.ReadFile(after)
	require.NoError(t, err)
	assert.Equal(t, beforeContents, afterContents)
}
