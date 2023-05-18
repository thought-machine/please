package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllAssetsReturnsAListOfBuildDefinitionFiles(t *testing.T) {
	assets, err := AllAssets()

	defs := []string{
		"builtins.build_defs",
		"config_rules.build_defs",
		"misc_rules.build_defs",
		"subrepo_rules.build_defs",
	}

	assert.NoError(t, err)
	assert.ElementsMatch(t, assets, defs)
}

func TestReadAssetReadsCorrectAsset(t *testing.T) {
	output, err := ReadAsset("builtins.build_defs")

	assert.NoError(t, err)
	assert.Contains(t, string(output), "def build_rule")
}

func TestReadAssetReturnsErrorIfFileDoesNotExist(t *testing.T) {
	output, err := ReadAsset("does-not-exist.txt")

	assert.Error(t, err)
	assert.Nil(t, output)
}
