package query

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

var repoRoot = os.Getenv("DATA")

func TestPackageToParseInRepoRoot(t *testing.T) {
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("failed to change to repo root: %v", err)
	}

	config := core.DefaultConfiguration()
	config.Parse.BuildFileName = []string{"BUILD_FILE"}

	t.Run("complete //", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//", ".")
		assert.Equal(t, "", toParse)

		require.Len(t, pkgs, 2)
		assert.ElementsMatch(t, []string{"foo", "bing"}, pkgs)
	})

	t.Run("complete //foo", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//foo", ".")
		assert.Equal(t, "foo", toParse)

		require.Len(t, pkgs, 2)
		assert.ElementsMatch(t, []string{"foo/bar", "foo/baz"}, pkgs)
	})

	t.Run("complete //foo/", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//foo/", ".")
		assert.Equal(t, "", toParse)

		require.Len(t, pkgs, 2)
		assert.ElementsMatch(t, []string{"foo/bar", "foo/baz"}, pkgs)
	})


	t.Run("complete //foo/bar/", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//foo/bar/", ".")
		assert.Equal(t, "foo/bar/net/thoughtmachine/please", toParse)

		require.Len(t, pkgs, 1)
		assert.ElementsMatch(t, []string{"foo/bar/net/thoughtmachine/please/main"}, pkgs)
	})

	t.Run("complete //bing/", func(t *testing.T) {
		pkgs, toParse := getPackagesAndPackageToParse(config, "//bing/", ".")
		assert.Equal(t, "bing/net/thoughtmachine/please", toParse)

		require.Len(t, pkgs, 0)
	})

}