package query

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

var repoRoot = os.Getenv("DATA")

func TestGetPackageToParse(t *testing.T) {
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("failed to change to repo root: %v", err)
	}

	config := core.DefaultConfiguration()
	config.Parse.BuildFileName = []string{"BUILD_FILE"}

	t.Run("complete in repo root", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//")
		assert.Equal(t, "", toParse)

		require.Len(t, pkgs, 2)
		assert.ElementsMatch(t, []string{"foo", "bing"}, pkgs)
	})

	t.Run("complete package with sub-packages", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//foo")
		assert.Equal(t, "foo", toParse)

		require.Len(t, pkgs, 2)
		assert.ElementsMatch(t, []string{"foo/bar", "foo/baz"}, pkgs)
	})

	t.Run("complete packages only", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//foo/")
		assert.Equal(t, "", toParse)

		require.Len(t, pkgs, 2)
		assert.ElementsMatch(t, []string{"foo/bar", "foo/baz"}, pkgs)
	})

	t.Run("complete labels only", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//foo:")
		assert.Equal(t, "", toParse)

		require.Len(t, pkgs, 0)
	})

	t.Run("complete package with single nested subpackage", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//bing/")
		assert.Equal(t, "bing/net/thoughtmachine/please", toParse)

		require.Len(t, pkgs, 0)
	})

	t.Run("complete package with nested subpackages", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//foo/bar/")
		assert.Equal(t, "foo/bar/net/thoughtmachine/please", toParse)

		require.Len(t, pkgs, 1)
		assert.ElementsMatch(t, []string{"foo/bar/net/thoughtmachine/please/main"}, pkgs)
	})
}
