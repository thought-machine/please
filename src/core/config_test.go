package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlzConfigWorking(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/working.plzconfig"})
	assert.NoError(t, err)
	assert.Equal(t, "pexmabob", config.Python.PexTool)
	assert.Equal(t, "javac", config.Java.JavacTool)
	assert.Equal(t, "8", config.Java.SourceLevel)
	assert.Equal(t, "7", config.Java.TargetLevel)
}

func TestPlzConfigFailing(t *testing.T) {
	_, err := ReadConfigFiles([]string{"src/core/test_data/failing.plzconfig"})
	assert.Error(t, err)
}

func TestMultiplePlzConfigFiles(t *testing.T) {
	config, err := ReadConfigFiles([]string{
		"src/core/test_data/working.plzconfig",
		"src/core/test_data/failing.plzconfig",
	})
	assert.Error(t, err)
	// Quick check of this - we should have still read the first config file correctly.
	assert.Equal(t, "javac", config.Java.JavacTool)
}

func TestConfigSlicesOverwrite(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/slices.plzconfig"})
	assert.NoError(t, err)
	// This should be completely overwritten by the config file
	assert.Equal(t, []string{"/sbin"}, config.Build.Path)
	// This should still get the defaults.
	assert.Equal(t, []string{"BUILD"}, config.Please.BuildFileName)
}
