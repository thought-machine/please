package core

import (
	"encoding/base64"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/scm"
)

// ExpandHomePath is an alias to the function in fs for compatibility.
var ExpandHomePath func(string) string = fs.ExpandHomePath

// A BuildEnv is a representation of the build environment that also knows how to log itself.
type BuildEnv []string

// GeneralBuildEnvironment creates the shell env vars used for a command, not based
// on any specific target etc.
func GeneralBuildEnvironment(config *Configuration) BuildEnv {
	// TODO(peterebden): why is this not just config.GetBuildEnv()?
	env := BuildEnv{
		// Need this for certain tools, for example sass
		"LANG=" + config.Build.Lang,
		// Expose the requested build config. We might also want to expose
		// the command that's actually running (although typically this is more useful,
		// because targets using this want to avoid defining different commands).
		"BUILD_CONFIG=" + config.Build.Config,
	}
	if config.Go.GoRoot != "" {
		env = append(env, "GOROOT="+config.Go.GoRoot)
	}
	if config.Cpp.PkgConfigPath != "" {
		env = append(env, "PKG_CONFIG_PATH="+config.Cpp.PkgConfigPath)
	}
	return append(env, config.GetBuildEnv()...)
}

// buildEnvironment returns the basic parts of the build environment.
func buildEnvironment(state *BuildState, target *BuildTarget) BuildEnv {
	env := append(GeneralBuildEnvironment(state.Config),
		"PKG="+target.Label.PackageName,
		"PKG_DIR="+target.Label.PackageDir(),
		"NAME="+target.Label.Name,
		"CONFIG="+state.Config.Build.Config,
	)
	if target.PassEnv != nil {
		for _, e := range *target.PassEnv {
			env = append(env, e+"="+os.Getenv(e))
		}
	}
	return env
}

// BuildEnvironment creates the shell env vars to be passed into the exec.Command calls made by plz.
// Note that we lie about the location of HOME in order to keep some tools happy.
// We read this as being slightly more POSIX-compliant than not having it set at all...
func BuildEnvironment(state *BuildState, target *BuildTarget, tmpDir string) BuildEnv {
	env := buildEnvironment(state, target)
	sources := target.AllSourcePaths(state.Graph)
	outEnv := target.GetTmpOutputAll(target.Outputs())
	abs := path.IsAbs(tmpDir)

	env = append(env,
		"TMP_DIR="+tmpDir,
		"TMPDIR="+tmpDir,
		"SRCS="+strings.Join(sources, " "),
		"OUTS="+strings.Join(outEnv, " "),
		"HOME="+tmpDir,
		"TOOLS="+strings.Join(toolPaths(state, target.Tools, abs), " "),
		// Set a consistent hash seed for Python. Important for build determinism.
		"PYTHONHASHSEED=42",
	)
	// The OUT variable is only available on rules that have a single output.
	if len(outEnv) == 1 {
		env = append(env, "OUT="+path.Join(tmpDir, outEnv[0]))
	}
	// The SRC variable is only available on rules that have a single source file.
	if len(sources) == 1 {
		env = append(env, "SRC="+sources[0])
	}
	// Similarly, TOOL is only available on rules with a single tool.
	if len(target.Tools) == 1 {
		env = append(env, "TOOL="+toolPath(state, target.Tools[0], abs))
	}
	// Named source groups if the target declared any.
	for name, srcs := range target.NamedSources {
		paths := target.SourcePaths(state.Graph, srcs)
		env = append(env, "SRCS_"+strings.ToUpper(name)+"="+strings.Join(paths, " "))
	}
	// Named output groups similarly.
	for name, outs := range target.DeclaredNamedOutputs() {
		outs = target.GetTmpOutputAll(outs)
		env = append(env, "OUTS_"+strings.ToUpper(name)+"="+strings.Join(outs, " "))
	}
	// Named tools as well.
	for name, tools := range target.namedTools {
		env = append(env, "TOOLS_"+strings.ToUpper(name)+"="+strings.Join(toolPaths(state, tools, abs), " "))
	}
	// Secrets, again only if they declared any.
	if len(target.Secrets) > 0 {
		secrets := "SECRETS=" + ExpandHomePath(strings.Join(target.Secrets, ":"))
		secrets = strings.Replace(secrets, ":", " ", -1)
		env = append(env, secrets)
	}
	// NamedSecrets, if they declared any.
	for name, secrets := range target.NamedSecrets {
		secrets := "SECRETS_" + strings.ToUpper(name) + "=" + ExpandHomePath(strings.Join(secrets, ":"))
		secrets = strings.Replace(secrets, ":", " ", -1)
		env = append(env, secrets)
	}
	if state.Config.Bazel.Compatibility {
		// Obviously this is only a subset of the variables Bazel would expose, but there's
		// no point populating ones that we literally have no clue what they should be.
		// To be honest I don't terribly like these, I'm pretty sure that using $GENDIR in
		// your genrule is not a good sign.
		env = append(env, "GENDIR="+path.Join(RepoRoot, GenDir))
		env = append(env, "BINDIR="+path.Join(RepoRoot, BinDir))
	}
	return env
}

// TestEnvironment creates the environment variables for a test.
func TestEnvironment(state *BuildState, target *BuildTarget, testDir string) BuildEnv {
	env := buildEnvironment(state, target)
	resultsFile := path.Join(testDir, TestResultsFile)
	env = append(env,
		"TEST_DIR="+testDir,
		"TMP_DIR="+testDir,
		"TMPDIR="+testDir,
		"TEST_ARGS="+strings.Join(state.TestArgs, ","),
		"RESULTS_FILE="+resultsFile,
		// We shouldn't really have specific things like this here, but it really is just easier to set it.
		"GTEST_OUTPUT=xml:"+resultsFile,
		"PEX_NOCACHE=true",
	)
	env = append(env, "HOME="+testDir)
	if state.NeedCoverage && !target.HasAnyLabel(state.Config.Test.DisableCoverage) {
		env = append(env,
			"COVERAGE=true",
			"COVERAGE_FILE="+path.Join(testDir, CoverageFile),
		)
	}
	if len(target.Outputs()) > 0 {
		// Bit of a hack; ideally we would be unaware of the sandbox here.
		if target.TestSandbox && runtime.GOOS == "linux" && !strings.HasPrefix(RepoRoot, "/tmp/") && testDir != "." {
			env = append(env, "TEST="+path.Join(SandboxDir, target.Outputs()[0]))
		} else {
			env = append(env, "TEST="+path.Join(testDir, target.Outputs()[0]))
		}
	}
	if len(target.Data) > 0 {
		env = append(env, "DATA="+strings.Join(target.AllDataPaths(state.Graph), " "))
	}
	if target.namedData != nil {
		for name, data := range target.namedData {
			paths := target.SourcePaths(state.Graph, data)
			env = append(env, "DATA_"+strings.ToUpper(name)+"="+strings.Join(paths, " "))
		}
	}
	// Bit of a hack for gcov which needs access to its .gcno files.
	if target.HasLabel("cc") {
		env = append(env, "GCNO_DIR="+path.Join(RepoRoot, GenDir, target.Label.PackageName))
	}
	if state.DebugTests {
		env = append(env, "DEBUG=true")
	}
	return env
}

// StampedBuildEnvironment returns the shell env vars to be passed into exec.Command.
// Optionally includes a stamp if the target is marked as such.
func StampedBuildEnvironment(state *BuildState, target *BuildTarget, stamp []byte, tmpDir string) BuildEnv {
	env := BuildEnvironment(state, target, tmpDir)
	if target.Stamp {
		stampEnvOnce.Do(initStampEnv)
		env = append(env, stampEnv...)
		env = append(env, "STAMP_FILE="+target.StampFileName())
		return append(env, "STAMP="+base64.RawURLEncoding.EncodeToString(stamp))
	}
	return env
}

// stampEnv is the generic (i.e. non-target-specific) environment variables we pass to a
// build rule marked with stamp=True.
var stampEnv BuildEnv
var stampEnvOnce sync.Once

func initStampEnv() {
	repoScm := scm.NewFallback(RepoRoot)
	revision := repoScm.CurrentRevIdentifier()
	stampEnv = BuildEnv{
		"SCM_COMMIT_DATE=" + repoScm.CurrentRevDate("20060102"),
		"SCM_REVISION=" + revision,
		"SCM_DESCRIBE=" + repoScm.DescribeIdentifier(revision),
	}
}

func toolPath(state *BuildState, tool BuildInput, abs bool) string {
	if label := tool.Label(); label != nil {
		path := state.Graph.TargetOrDie(*label).toolPath(abs)
		if !strings.Contains(path, "/") {
			path = "./" + path
		}
		return path
	} else if path := tool.Paths(state.Graph)[0]; abs || !strings.HasPrefix(path, state.Config.PleaseLocation) {
		return path
	}
	return tool.LocalPaths(state.Graph)[0]
}

func toolPaths(state *BuildState, tools []BuildInput, abs bool) []string {
	ret := make([]string, len(tools))
	for i, tool := range tools {
		ret[i] = toolPath(state, tool, abs)
	}
	return ret
}

// ReplaceEnvironment is a function suitable for passing to os.Expand to replace environment
// variables from this BuildEnv.
func (env BuildEnv) ReplaceEnvironment(s string) string {
	for _, e := range env {
		if strings.HasPrefix(e, s+"=") {
			return e[len(s)+1:]
		}
	}
	return ""
}

// Replace replaces the value of the given variable in this BuildEnv.
func (env BuildEnv) Replace(key, value string) {
	key = key + "="
	for i, e := range env {
		if strings.HasPrefix(e, key) {
			env[i] = key + value
		}
	}
}

// Redacted implements the interface for our logging implementation.
func (env BuildEnv) Redacted() interface{} {
	r := make(BuildEnv, len(env))
	for i, e := range env {
		r[i] = e
		split := strings.SplitN(e, "=", 2)
		if len(split) == 2 && (strings.Contains(split[0], "SECRET") || strings.Contains(split[0], "PASSWORD")) {
			r[i] = split[0] + "=" + "************"
		}
	}
	return r
}

// String implements the fmt.Stringer interface
func (env BuildEnv) String() string {
	return strings.Join(env, "\n")
}
