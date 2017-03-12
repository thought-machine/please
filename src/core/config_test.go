package core

import (
	"testing"
	"time"

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

func TestConfigOverrideDuration(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"build.timeout": "15"})
	assert.NoError(t, err)
	assert.EqualValues(t, 15*time.Second, config.Build.Timeout)
}

func TestConfigOverrideNonIntDuration(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"build.timeout": "10m"})
	assert.NoError(t, err)
	assert.EqualValues(t, 10*time.Minute, config.Build.Timeout)
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

func TestConfigOverrideLabelSlice(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"gc.keep": "//src/core:core"})
	assert.NoError(t, err)
	assert.Equal(t, []BuildLabel{ParseBuildLabel("//src/core:core", "")}, config.Gc.Keep)
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

func TestDynamicSection(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/aliases.plzconfig"})
	assert.NoError(t, err)
	expected := map[string]string{
		"deploy":      "run //deployment:deployer --",
		"deploy dev":  "run //deployment:deployer -- --env=dev",
		"deploy prod": "run //deployment:deployer -- --env=prod",
	}
	assert.Equal(t, expected, config.Aliases)
}

func TestDynamicSubsection(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/metrics.plzconfig"})
	assert.NoError(t, err)
	assert.EqualValues(t, "http://localhost:9091", config.Metrics.PushGatewayURL)
	expected := map[string]string{
		"branch": "git rev-parse --abbrev-ref HEAD",
	}
	assert.Equal(t, expected, config.CustomMetricLabels)
}

func TestReadSemver(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/version_good.plzconfig"})
	assert.NoError(t, err)
	assert.EqualValues(t, 2, config.Please.Version.Major)
	assert.EqualValues(t, 3, config.Please.Version.Minor)
	assert.EqualValues(t, 4, config.Please.Version.Patch)
	config, err = ReadConfigFiles([]string{"src/core/test_data/version_bad.plzconfig"})
	assert.Error(t, err)
}

func TestReadDurations(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/duration_good.plzconfig"})
	assert.NoError(t, err)
	assert.EqualValues(t, 500*time.Millisecond, config.Metrics.PushTimeout)
	assert.EqualValues(t, 5*time.Second, config.Metrics.PushFrequency)
	config, err = ReadConfigFiles([]string{"src/core/test_data/duration_bad.plzconfig"})
	assert.Error(t, err)
}

func TestReadByteSizes(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/bytesize_good.plzconfig"})
	assert.NoError(t, err)
	assert.EqualValues(t, 500*1000*1000, config.Cache.RpcMaxMsgSize)
	config, err = ReadConfigFiles([]string{"src/core/test_data/bytesize_bad.plzconfig"})
	assert.Error(t, err)
}

func TestReadContainers(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/container_good.plzconfig"})
	assert.NoError(t, err)
	assert.EqualValues(t, TestContainerDocker, config.Test.DefaultContainer)
	config, err = ReadConfigFiles([]string{"src/core/test_data/container_bad.plzconfig"})
	assert.Error(t, err)
}
