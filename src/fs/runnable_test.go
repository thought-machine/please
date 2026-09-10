package fs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplainUnrunnableSaysNothingAboutMissingFiles(t *testing.T) {
	assert.Empty(t, ExplainUnrunnable(filepath.Join(t.TempDir(), "nothing-here")))
	assert.Empty(t, ExplainUnrunnable(""))
	assert.Empty(t, ExplainUnrunnable(t.TempDir()), "a directory isn't a binary that failed to run")
}

func TestExplainUnrunnableSaysNothingAboutProperlyNamedFiles(t *testing.T) {
	file := filepath.Join(t.TempDir(), "tool"+ExeSuffix)
	require.NoError(t, os.WriteFile(file, nil, 0o755))
	assert.Empty(t, ExplainUnrunnable(file), "this one is named the way the platform wants")
}

func TestExplainUnrunnableNamesTheSuffix(t *testing.T) {
	file := filepath.Join(t.TempDir(), "tool")
	require.NoError(t, os.WriteFile(file, nil, 0o755))
	explanation := ExplainUnrunnable(file)
	if runtime.GOOS != "windows" {
		// The executable bit decides on Unix, and the error already says so.
		assert.Empty(t, explanation)
		return
	}
	assert.Contains(t, explanation, "tool.exe")
}
