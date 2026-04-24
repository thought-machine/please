package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

const testDir = "src/format/test_data"

func TestFormat(t *testing.T) {
	files, err := os.ReadDir(testDir)
	assert.NoError(t, err)

	patchConfigByFile := map[string]func(*core.Configuration) {
		"shorten_labels":  func(c *core.Configuration) {
			c.Format.ShortenAbsoluteLabelsToRelative = true
		},
	}

	for _, file := range files {
		if test, isBefore := strings.CutSuffix(file.Name(), ".before.build"); isBefore {
			t.Run(test, func(t *testing.T) {
				before := filepath.Join(testDir, test+".before.build")
				after := filepath.Join(testDir, test+".after.build")

				config := core.DefaultConfiguration()
				if patchConfig := patchConfigByFile[test]; patchConfig != nil {
					patchConfig(config)
				}

				changed, err := Format(config, []string{before}, false, true)
				assert.NoError(t, err)
				assert.True(t, changed)

				// N.B. this rewrites the file; be careful if you're adding more tests here.
				changed, err = Format(config, []string{before}, true, false)
				assert.NoError(t, err)
				assert.True(t, changed)

				beforeContents, err := os.ReadFile(before)
				require.NoError(t, err)
				afterContents, err := os.ReadFile(after)
				require.NoError(t, err)
				assert.Equal(t, string(afterContents), string(beforeContents))
			})
		}
	}
}
