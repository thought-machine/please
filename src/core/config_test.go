package core

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"cli"
)

func TestPlzConfigWorking(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/working.plzconfig"}, "")
	assert.NoError(t, err)
	assert.Equal(t, "pexmabob", config.Python.PexTool)
	assert.Equal(t, "javac", config.Java.JavacTool)
	assert.Equal(t, "jlink", config.Java.JlinkTool)
	assert.Equal(t, "/path/to/java/home", config.Java.JavaHome)
	assert.Equal(t, "8", config.Java.SourceLevel)
	assert.Equal(t, "7", config.Java.TargetLevel)
}

func TestPlzConfigFailing(t *testing.T) {
	_, err := ReadConfigFiles([]string{"src/core/test_data/failing.plzconfig"}, "")
	assert.Error(t, err)
}

func TestPlzConfigProfile(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/working.plzconfig"}, "dev")
	assert.NoError(t, err)
	assert.Equal(t, "pexmabob", config.Python.PexTool)
	assert.Equal(t, "/opt/java/bin/javac", config.Java.JavacTool)
	assert.Equal(t, "8", config.Java.SourceLevel)
	assert.Equal(t, "8", config.Java.TargetLevel)
}

func TestMultiplePlzConfigFiles(t *testing.T) {
	config, err := ReadConfigFiles([]string{
		"src/core/test_data/working.plzconfig",
		"src/core/test_data/failing.plzconfig",
	}, "")
	assert.Error(t, err)
	// Quick check of this - we should have still read the first config file correctly.
	assert.Equal(t, "javac", config.Java.JavacTool)
}

func TestConfigSlicesOverwrite(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/slices.plzconfig"}, "")
	assert.NoError(t, err)
	// This should be completely overwritten by the config file
	assert.Equal(t, []string{"/sbin"}, config.Build.Path)
	// This should still get the defaults.
	assert.Equal(t, []string{"BUILD"}, config.Parse.BuildFileName)
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
	assert.True(t, config.Cache.RPCWriteable)
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

func TestConfigOverrideURLSlice(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"java.defaultmavenrepo": "https://repo1.maven.org,https://maven.google.com"})
	assert.NoError(t, err)
	assert.Equal(t, []cli.URL{"https://repo1.maven.org", "https://maven.google.com"}, config.Java.DefaultMavenRepo)
}

func TestConfigOverrideMap(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{
		"buildconfig.android-keystore":          "/tmp/debug.key",
		"buildconfig.android-keystore-password": "android",
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"android-keystore":          "/tmp/debug.key",
		"android-keystore-password": "android",
	}, config.BuildConfig)
}

func TestConfigOverrideUnknownName(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"build.blah": "whatevs"})
	assert.Error(t, err)
}

func TestConfigOverrideURL(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"metrics.pushgatewayurl": "http://gateway:9091"})
	assert.NoError(t, err)
	assert.EqualValues(t, "http://gateway:9091", config.Metrics.PushGatewayURL)
}

func TestConfigOverrideOptions(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"python.testrunner": "pytest"})
	assert.NoError(t, err)
	assert.Equal(t, "pytest", config.Python.TestRunner)
	err = config.ApplyOverrides(map[string]string{"python.testrunner": "junit"})
	assert.Error(t, err)
}

func TestDynamicSection(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/aliases.plzconfig"}, "")
	assert.NoError(t, err)
	expected := map[string]string{
		"deploy":      "run //deployment:deployer --",
		"deploy dev":  "run //deployment:deployer -- --env=dev",
		"deploy prod": "run //deployment:deployer -- --env=prod",
	}
	assert.Equal(t, expected, config.Aliases)
}

func TestDynamicSubsection(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/metrics.plzconfig"}, "")
	assert.NoError(t, err)
	assert.EqualValues(t, "http://localhost:9091", config.Metrics.PushGatewayURL)
	expected := map[string]string{
		"branch": "git rev-parse --abbrev-ref HEAD",
	}
	assert.Equal(t, expected, config.CustomMetricLabels)
}

func TestReadSemver(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/version_good.plzconfig"}, "")
	assert.NoError(t, err)
	assert.EqualValues(t, 2, config.Please.Version.Major)
	assert.EqualValues(t, 3, config.Please.Version.Minor)
	assert.EqualValues(t, 4, config.Please.Version.Patch)
	config, err = ReadConfigFiles([]string{"src/core/test_data/version_bad.plzconfig"}, "")
	assert.Error(t, err)
}

func TestReadDurations(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/duration_good.plzconfig"}, "")
	assert.NoError(t, err)
	assert.EqualValues(t, 500*time.Millisecond, config.Metrics.PushTimeout)
	assert.EqualValues(t, 5*time.Second, config.Metrics.PushFrequency)
	config, err = ReadConfigFiles([]string{"src/core/test_data/duration_bad.plzconfig"}, "")
	assert.Error(t, err)
}

func TestReadByteSizes(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/bytesize_good.plzconfig"}, "")
	assert.NoError(t, err)
	assert.EqualValues(t, 500*1000*1000, config.Cache.RPCMaxMsgSize)
	config, err = ReadConfigFiles([]string{"src/core/test_data/bytesize_bad.plzconfig"}, "")
	assert.Error(t, err)
}

func TestReadContainers(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/container_good.plzconfig"}, "")
	assert.NoError(t, err)
	assert.EqualValues(t, ContainerImplementationDocker, config.Test.DefaultContainer)
	config, err = ReadConfigFiles([]string{"src/core/test_data/container_bad.plzconfig"}, "")
	assert.Error(t, err)
}

func TestCompletions(t *testing.T) {
	config := DefaultConfiguration()
	completions := config.Completions("python.pip")
	assert.Equal(t, 2, len(completions))
	assert.Equal(t, "python.piptool:", completions[0].Item)
	assert.Equal(t, "python.pipflags:", completions[1].Item)
}

func TestConfigVerifiesOptions(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/testrunner_good.plzconfig"}, "")
	assert.NoError(t, err)
	assert.Equal(t, "pytest", config.Python.TestRunner)
	_, err = ReadConfigFiles([]string{"src/core/test_data/testrunner_bad.plzconfig"}, "")
	assert.Error(t, err)
}

func TestBuildEnvSection(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/buildenv.plzconfig"}, "")
	assert.NoError(t, err)
	expected := []string{
		"ARCH=" + runtime.GOARCH,
		"BAR_BAR=first",
		"FOO_BAR=second",
		"OS=" + runtime.GOOS,
		"XARCH=x86_64",
		"XOS=" + xos(),
	}
	assert.Equal(t, expected, config.GetBuildEnv())
}

func TestPassEnv(t *testing.T) {
	err := os.Setenv("FOO", "first")
	assert.NoError(t, err)
	err = os.Setenv("BAR", "second")
	assert.NoError(t, err)
	config, err := ReadConfigFiles([]string{"src/core/test_data/passenv.plzconfig"}, "")
	assert.NoError(t, err)
	expected := []string{
		"ARCH=" + runtime.GOARCH,
		"BAR=second",
		"FOO=first",
		"OS=" + runtime.GOOS,
		"XARCH=x86_64",
		"XOS=" + xos(),
	}
	assert.Equal(t, expected, config.GetBuildEnv())
}

func xos() string {
	if runtime.GOOS == "darwin" {
		return "osx"
	}
	return runtime.GOOS
}
