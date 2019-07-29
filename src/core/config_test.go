package core

import (
	"bytes"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/cli"
	"reflect"
	"strings"
)

func TestPlzConfigWorking(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/working.plzconfig"}, nil)
	assert.NoError(t, err)
	assert.Equal(t, "pexmabob", config.Python.PexTool)
	assert.Equal(t, "javac", config.Java.JavacTool)
	assert.Equal(t, "jlink", config.Java.JlinkTool)
	assert.Equal(t, "/path/to/java/home", config.Java.JavaHome)
	assert.Equal(t, "8", config.Java.SourceLevel)
	assert.Equal(t, "7", config.Java.TargetLevel)
	assert.Equal(t, "10", config.Java.ReleaseLevel)
}

func TestPlzConfigFailing(t *testing.T) {
	_, err := ReadConfigFiles([]string{"src/core/test_data/failing.plzconfig"}, nil)
	assert.Error(t, err)
}

func TestPlzConfigProfile(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/working.plzconfig"}, []string{"dev"})
	assert.NoError(t, err)
	assert.Equal(t, "pexmabob", config.Python.PexTool)
	assert.Equal(t, "/opt/java/bin/javac", config.Java.JavacTool)
	assert.Equal(t, "8", config.Java.SourceLevel)
	assert.Equal(t, "8", config.Java.TargetLevel)
	assert.Equal(t, "10", config.Java.ReleaseLevel)
}

func TestMultiplePlzConfigFiles(t *testing.T) {
	config, err := ReadConfigFiles([]string{
		"src/core/test_data/working.plzconfig",
		"src/core/test_data/failing.plzconfig",
	}, nil)
	assert.Error(t, err)
	// Quick check of this - we should have still read the first config file correctly.
	assert.Equal(t, "javac", config.Java.JavacTool)
}

func TestConfigSlicesOverwrite(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/slices.plzconfig"}, nil)
	assert.NoError(t, err)
	// This should be completely overwritten by the config file
	assert.Equal(t, []string{"/sbin"}, config.Build.Path)
	// This should still get the defaults.
	assert.Equal(t, []string{"BUILD", "BUILD.plz"}, config.Parse.BuildFileName)
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
	err := config.ApplyOverrides(map[string]string{"test.upload": "http://gateway:9091"})
	assert.NoError(t, err)
	assert.EqualValues(t, "http://gateway:9091", config.Test.Upload)
}

func TestConfigOverrideOptions(t *testing.T) {
	config := DefaultConfiguration()
	err := config.ApplyOverrides(map[string]string{"python.testrunner": "pytest"})
	assert.NoError(t, err)
	assert.Equal(t, "pytest", config.Python.TestRunner)
	err = config.ApplyOverrides(map[string]string{"python.testrunner": "junit"})
	assert.Error(t, err)
}

func TestReadSemver(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/version_good.plzconfig"}, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, config.Please.Version.Major)
	assert.EqualValues(t, 3, config.Please.Version.Minor)
	assert.EqualValues(t, 4, config.Please.Version.Patch)
	config, err = ReadConfigFiles([]string{"src/core/test_data/version_bad.plzconfig"}, nil)
	assert.Error(t, err)
}

func TestReadDurations(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/duration_good.plzconfig"}, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, 500*time.Millisecond, config.Build.Timeout)
	assert.EqualValues(t, 5*time.Second, config.Test.Timeout)
	config, err = ReadConfigFiles([]string{"src/core/test_data/duration_bad.plzconfig"}, nil)
	assert.Error(t, err)
}

func TestReadByteSizes(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/bytesize_good.plzconfig"}, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, 500*1000*1000, config.Cache.RPCMaxMsgSize)
	config, err = ReadConfigFiles([]string{"src/core/test_data/bytesize_bad.plzconfig"}, nil)
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
	config, err := ReadConfigFiles([]string{"src/core/test_data/testrunner_good.plzconfig"}, nil)
	assert.NoError(t, err)
	assert.Equal(t, "pytest", config.Python.TestRunner)
	_, err = ReadConfigFiles([]string{"src/core/test_data/testrunner_bad.plzconfig"}, nil)
	assert.Error(t, err)
}

func TestBuildEnvSection(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/buildenv.plzconfig"}, nil)
	assert.NoError(t, err)
	expected := []string{
		"ARCH=" + runtime.GOARCH,
		"BAR_BAR=first",
		"FOO_BAR=second",
		"GOARCH=" + runtime.GOARCH,
		"GOOS=" + runtime.GOOS,
		"OS=" + runtime.GOOS,
		"PATH=" + os.Getenv("TMP_DIR") + ":/usr/local/bin:/usr/bin:/bin",
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
	config, err := ReadConfigFiles([]string{"src/core/test_data/passenv.plzconfig"}, nil)
	assert.NoError(t, err)
	expected := []string{
		"ARCH=" + runtime.GOARCH,
		"BAR=second",
		"FOO=first",
		"GOARCH=" + runtime.GOARCH,
		"GOOS=" + runtime.GOOS,
		"OS=" + runtime.GOOS,
		"PATH=" + os.Getenv("TMP_DIR") + ":" + os.Getenv("PATH"),
		"XARCH=x86_64",
		"XOS=" + xos(),
	}
	assert.Equal(t, expected, config.GetBuildEnv())
}

func TestBuildPathWithPathEnv(t *testing.T) {
	config, err := ReadConfigFiles([]string{"src/core/test_data/passenv.plzconfig"}, nil)
	assert.NoError(t, err)
	assert.Equal(t, config.Build.Path, strings.Split(os.Getenv("PATH"), ":"))
}

func xos() string {
	if runtime.GOOS == "darwin" {
		return "osx"
	}
	return runtime.GOOS
}

func TestUpdateArgsWithAliases(t *testing.T) {
	c := DefaultConfiguration()
	c.Alias = map[string]*Alias{
		"deploy": {Cmd: "run //deploy:deployer --"},
		"mytool": {Cmd: "run //mytool:tool --"},
	}

	args := c.UpdateArgsWithAliases([]string{"plz", "run", "//src/tools:tool"})
	assert.EqualValues(t, []string{"plz", "run", "//src/tools:tool"}, args)

	args = c.UpdateArgsWithAliases([]string{"plz", "deploy", "something"})
	assert.EqualValues(t, []string{"plz", "run", "//deploy:deployer", "--", "something"}, args)

	args = c.UpdateArgsWithAliases([]string{"plz", "mytool"})
	assert.EqualValues(t, []string{"plz", "run", "//mytool:tool", "--"}, args)

	args = c.UpdateArgsWithAliases([]string{"plz", "mytool", "deploy", "something"})
	assert.EqualValues(t, []string{"plz", "run", "//mytool:tool", "--", "deploy", "something"}, args)
}

func TestUpdateArgsWithQuotedAliases(t *testing.T) {
	c := DefaultConfiguration()
	c.Alias = map[string]*Alias{
		"release": {Cmd: "build -o 'buildconfig.gpg_userid:Please Releases <releases@please.build>' //package:tarballs"},
	}
	args := c.UpdateArgsWithAliases([]string{"plz", "release"})
	assert.EqualValues(t, []string{"plz", "build", "-o", "buildconfig.gpg_userid:Please Releases <releases@please.build>", "//package:tarballs"}, args)
}

func TestParseNewFormatAliases(t *testing.T) {
	c, err := ReadConfigFiles([]string{"src/core/test_data/alias.plzconfig"}, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(c.Alias))
	a := c.Alias["auth"]
	assert.Equal(t, "run //infra:auth --", a.Cmd)
	assert.EqualValues(t, []string{"gcp", "aws k8s", "aws ecr"}, a.Subcommand)
	assert.EqualValues(t, []string{"--host", "--repo"}, a.Flag)
}

func TestAttachAliasFlags(t *testing.T) {
	c, err := ReadConfigFiles([]string{"src/core/test_data/alias.plzconfig"}, nil)
	assert.NoError(t, err)
	os.Setenv("GO_FLAGS_COMPLETION", "1")
	p := flags.NewParser(&struct{}{}, 0)
	b := c.AttachAliasFlags(p)
	assert.True(t, b)
	completions := []string{}
	p.CompletionHandler = func(items []flags.Completion) {
		completions = make([]string, len(items))
		for i, item := range items {
			completions[i] = item.Item
		}
	}

	_, err = p.ParseArgs([]string{"plz", "au"})
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"auth"}, completions)

	_, err = p.ParseArgs([]string{"plz", "auth", "gc"})
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"gcp"}, completions)

	_, err = p.ParseArgs([]string{"plz", "auth", "aws", "e"})
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"ecr"}, completions)

	_, err = p.ParseArgs([]string{"plz", "auth", "aws", "--h"})
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"--host"}, completions)

	_, err = p.ParseArgs([]string{"plz", "query", "ow"})
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"owners"}, completions)
}

func TestPrintAliases(t *testing.T) {
	c, err := ReadConfigFiles([]string{"src/core/test_data/alias.plzconfig"}, nil)
	assert.NoError(t, err)
	var buf bytes.Buffer
	c.PrintAliases(&buf)
	assert.Equal(t, `
Available commands for this repository:
  auth          Authenticates you.
  query owners  Queries owners of a thing.
`, buf.String())
}

func TestGetTags(t *testing.T) {
	config := DefaultConfiguration()
	tags := config.TagsToFields()

	assert.Equal(t, "Version", tags["PLZ_VERSION"].Name)
	assert.True(t, tags["PLZ_VERSION"].Type == reflect.TypeOf(cli.Version{}))
}
