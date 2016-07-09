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

func TestConfigOverrideString(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"python.pextool": "pexinator"})
	assert.NoError(t, err)
	assert.Equal(t, "pexinator", config.Python.PexTool)
}

func TestConfigOverrideUppercase(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"Python.PexTool": "pexinator"})
	assert.NoError(t, err)
	assert.Equal(t, "pexinator", config.Python.PexTool)
}

func TestConfigOverrideInt(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"build.timeout": "15"})
	assert.NoError(t, err)
	assert.Equal(t, 15, config.Build.Timeout)
}

func TestConfigOverrideBool(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"cache.rpcwriteable": "yes"})
	assert.NoError(t, err)
	assert.True(t, config.Cache.RpcWriteable)
}

func TestConfigOverrideSlice(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"build.path": "/mnt/bin,/mnt/sbin"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"/mnt/bin", "/mnt/sbin"}, config.Build.Path)
}

func TestConfigOverrideMap(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"aliases.blah": "run //blah:blah"})
	assert.Error(t, err, "Can't override map fields, but should handle it nicely.")
}

func TestConfigOverrideUnknownName(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"build.blah": "whatevs"})
	assert.Error(t, err)
}
